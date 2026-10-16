// Package indexer 在文档落库后异步重建 Qdrant 向量索引。
package indexer

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/Bluore/mutidoc/app/document/model"
	"github.com/zeromicro/go-zero/core/logx"
)

const (
	// defaultTimeout 单次重建(含向量化)的整体超时
	defaultTimeout = 60 * time.Second
	// embedBatch 单次向量化的文本条数
	embedBatch = 32
)

// Config 索引器配置
type Config struct {
	Chunk   model.ChunkConf
	Timeout time.Duration
}

// Indexer 按文档重建向量索引。
// 同一文档同时只跑一个重建任务, 期间到来的新请求会合并为一次补跑, 且补跑时重新
// 读取最新正文, 因此最终索引的一定是最后一次落库的内容。
type Indexer struct {
	documents model.DocumentsModel
	chunks    model.DocumentChunksModel
	embedder  model.Embedder
	config    Config

	mu      sync.Mutex
	running map[string]bool
	dirty   map[string]bool
}

// New 创建索引器, embedder 负责把分块文本向量化
func New(
	documents model.DocumentsModel,
	chunks model.DocumentChunksModel,
	embedder model.Embedder,
	config Config,
) *Indexer {
	if config.Timeout <= 0 {
		config.Timeout = defaultTimeout
	}
	return &Indexer{
		documents: documents,
		chunks:    chunks,
		embedder:  embedder,
		config:    config,
		running:   make(map[string]bool),
		dirty:     make(map[string]bool),
	}
}

// Schedule 异步重建文档索引, 不阻塞调用方, 失败只记日志
func (i *Indexer) Schedule(docID string) {
	if i == nil || docID == "" {
		return
	}

	i.mu.Lock()
	if i.running[docID] {
		i.dirty[docID] = true
		i.mu.Unlock()
		return
	}
	i.running[docID] = true
	i.mu.Unlock()

	go i.loop(docID)
}

// loop 串行执行该文档的重建, 期间积累的请求(脏标记)会再跑一轮, 跑完没人再要求才退出
func (i *Indexer) loop(docID string) {
	for {
		i.reindexDocument(docID)

		i.mu.Lock()
		if i.dirty[docID] {
			// 本轮执行期间又有落库, 再跑一轮; 清掉标记而不是置 false, 避免遗留空条目
			delete(i.dirty, docID)
			i.mu.Unlock()
			continue
		}
		delete(i.running, docID)
		i.mu.Unlock()
		return
	}
}

// reindexDocument 读取最新正文并重建该文档的全部分块; 文档已删除时清理历史分块
func (i *Indexer) reindexDocument(docID string) {
	ctx, cancel := context.WithTimeout(context.Background(), i.config.Timeout)
	defer cancel()

	detail, err := i.documents.FindOne(ctx, docID)
	if err != nil && !errors.Is(err, model.ErrNotFound) {
		logx.Errorw("reindex document: find document fail",
			logx.Field("error", err),
			logx.Field("document_id", docID),
		)
		return
	}
	if detail == nil {
		if err := i.chunks.DeleteByDocID(ctx, docID); err != nil {
			logx.Errorw("reindex document: delete chunks of removed document fail",
				logx.Field("error", err),
				logx.Field("document_id", docID),
			)
		}
		return
	}

	if err := i.build(ctx, detail); err != nil {
		logx.Errorw("reindex document fail",
			logx.Field("error", err),
			logx.Field("document_id", docID),
		)
	}
}

// build 把正文切块, 复用未变分块的向量, 只向量化新增内容, 再写入向量库
func (i *Indexer) build(ctx context.Context, detail *model.Documents) error {
	existing, err := i.chunks.FindByDocID(ctx, detail.Id)
	if err != nil {
		return err
	}

	reused := make(map[string]model.DocumentChunk, len(existing))
	maxIndex := int64(-1)
	for _, chunk := range existing {
		if chunk.ChunkIndex > maxIndex {
			maxIndex = chunk.ChunkIndex
		}
		if chunk.Text == "" || len(chunk.Vector) == 0 {
			continue
		}
		if _, ok := reused[chunk.Text]; !ok {
			reused[chunk.Text] = chunk
		}
	}

	texts := model.SplitDocumentChunks(detail.Content, i.config.Chunk.Size, i.config.Chunk.Overlap)
	now := time.Now().UTC()

	chunks := make([]model.DocumentChunk, 0, len(texts))
	pending := make([]string, 0, len(texts))
	pendingAt := make([]int, 0, len(texts))
	for index, text := range texts {
		chunk := model.DocumentChunk{
			DocID:      detail.Id,
			OwnerID:    detail.OwnerId,
			ChunkIndex: int64(index),
			Text:       text,
			CreateAt:   now,
		}
		if old, ok := reused[text]; ok {
			// 内容未变, 直接复用原向量与原创建时间
			chunk.Vector = old.Vector
			chunk.CreateAt = old.CreateAt
		} else {
			pending = append(pending, text)
			pendingAt = append(pendingAt, index)
		}
		chunks = append(chunks, chunk)
	}

	if err := i.embed(ctx, chunks, pending, pendingAt, now); err != nil {
		return err
	}
	if err := i.chunks.Upsert(ctx, chunks); err != nil {
		return err
	}
	// 正文变短时删掉多余的尾部, 避免过时文本留在索引里
	if maxIndex >= int64(len(chunks)) {
		if err := i.chunks.DeleteFrom(ctx, detail.Id, int64(len(chunks))); err != nil {
			return err
		}
	}

	logx.Infow("reindex document chunks success",
		logx.Field("document_id", detail.Id),
		logx.Field("chunks", len(chunks)),
		logx.Field("embedded", len(pending)),
	)
	return nil
}

// embed 分批向量化待处理文本, 并把结果写回对应分块
func (i *Indexer) embed(
	ctx context.Context,
	chunks []model.DocumentChunk,
	pending []string,
	pendingAt []int,
	createAt time.Time,
) error {
	for start := 0; start < len(pending); start += embedBatch {
		end := start + embedBatch
		if end > len(pending) {
			end = len(pending)
		}

		vectors, err := i.embedder.Embed(ctx, pending[start:end])
		if err != nil {
			return fmt.Errorf("embed chunks: %w", err)
		}
		if len(vectors) != end-start {
			return fmt.Errorf("embed chunks: got %d vectors for %d texts", len(vectors), end-start)
		}

		for offset, vector := range vectors {
			chunk := &chunks[pendingAt[start+offset]]
			chunk.Vector = vector
			chunk.CreateAt = createAt
		}
	}
	return nil
}
