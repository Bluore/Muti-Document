package model

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	// DefaultChunkSize 默认每个分块的字符数(按 rune 计)
	DefaultChunkSize = 500
	// DefaultChunkOverlap 默认相邻分块的重叠字符数, 即每块额外带上的上一块尾部内容
	DefaultChunkOverlap = 100

	// chunkNamespace 是文档分块点 ID 的 UUIDv5 命名空间, 保证同一文档同一序号 ID 稳定
	chunkNamespace = "8f0d7c8f-2f2b-4b8a-9d8f-0f2c9a4f1c31"
)

// ChunkConf 文档切块配置
type ChunkConf struct {
	Size    int
	Overlap int
}

// normalized 补齐缺省值, 并保证重叠小于分块长度
func (c ChunkConf) normalized() (size, overlap int) {
	size = c.Size
	if size <= 0 {
		size = DefaultChunkSize
	}
	overlap = c.Overlap
	if overlap < 0 {
		overlap = DefaultChunkOverlap
	}
	if overlap >= size {
		overlap = size / 5
	}
	return size, overlap
}

// Embedder 把文本向量化, 由 pkg/embedding.Client 实现
type Embedder interface {
	// Embed 批量向量化, 返回的向量与入参顺序一致
	Embed(ctx context.Context, texts []string) ([][]float32, error)
	// EmbedOne 向量化单条文本
	EmbedOne(ctx context.Context, text string) ([]float32, error)
}

// DocumentChunk 是 document_chunks 集合中的一条文档分块
type DocumentChunk struct {
	DocID      string    `json:"doc_id"`
	OwnerID    string    `json:"owner_id"`
	ChunkIndex int64     `json:"chunk_index"`
	Text       string    `json:"text"`
	CreateAt   time.Time `json:"create_at"`
	// Score 是向量检索的相似度, 仅检索结果返回
	Score float32 `json:"score,omitempty"`
	// Vector 只用于重建索引时复用, 不对外序列化
	Vector []float32 `json:"-"`
}

// DocumentChunkFilter 检索过滤条件, 零值表示不限制
type DocumentChunkFilter struct {
	DocID   string
	OwnerID string
}

// SplitDocumentChunks 把正文按固定长度切块。
// 每块向前多带 overlap 个字符(上一块的尾部边界内容), 让相邻分块在检索时保留上下文。
func SplitDocumentChunks(text string, size, overlap int) []string {
	size, overlap = ChunkConf{Size: size, Overlap: overlap}.normalized()

	runes := []rune(strings.TrimSpace(text))
	if len(runes) == 0 {
		return nil
	}

	step := size - overlap
	chunks := make([]string, 0, len(runes)/step+1)
	for start := 0; start < len(runes); start += step {
		end := start + size
		if end > len(runes) {
			end = len(runes)
		}
		if chunk := strings.TrimSpace(string(runes[start:end])); chunk != "" {
			chunks = append(chunks, chunk)
		}
		if end >= len(runes) {
			break
		}
	}
	return chunks
}

// documentChunkPointID 由文档 ID 与分块序号生成稳定点 ID(Qdrant 只接受 UUID / uint64)
func documentChunkPointID(docID string, chunkIndex int64) string {
	name := uuid.NewSHA1(uuid.MustParse(chunkNamespace), []byte(docID+"#"+strconv.FormatInt(chunkIndex, 10)))
	return name.String()
}
