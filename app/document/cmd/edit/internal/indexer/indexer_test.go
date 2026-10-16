package indexer

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/Bluore/mutidoc/app/document/model"
)

// fakeDocuments 只实现 FindOne, 其余方法不会被索引器调用
type fakeDocuments struct {
	model.DocumentsModel
	detail *model.Documents
	err    error
}

func (f fakeDocuments) FindOne(ctx context.Context, id string) (*model.Documents, error) {
	return f.detail, f.err
}

type fakeChunks struct {
	existing   []model.DocumentChunk
	upserted   []model.DocumentChunk
	deletedAll bool
	deletedAt  []int64
}

func (f *fakeChunks) DeleteByDocID(ctx context.Context, docID string) error {
	f.deletedAll = true
	return nil
}

func (f *fakeChunks) DeleteFrom(ctx context.Context, docID string, fromIndex int64) error {
	f.deletedAt = append(f.deletedAt, fromIndex)
	return nil
}

func (f *fakeChunks) FindByDocID(ctx context.Context, docID string) ([]model.DocumentChunk, error) {
	return f.existing, nil
}

func (f *fakeChunks) Upsert(ctx context.Context, chunks []model.DocumentChunk) error {
	f.upserted = chunks
	return nil
}

func (f *fakeChunks) Search(
	ctx context.Context,
	vector []float32,
	topK int,
	filter model.DocumentChunkFilter,
) ([]model.DocumentChunk, error) {
	return nil, nil
}

type fakeEmbedder struct {
	embedded []string
}

func (f *fakeEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	f.embedded = append(f.embedded, texts...)
	vectors := make([][]float32, len(texts))
	for i, text := range texts {
		vectors[i] = []float32{float32(len([]rune(text)))}
	}
	return vectors, nil
}

func (f *fakeEmbedder) EmbedOne(ctx context.Context, text string) ([]float32, error) {
	vectors, err := f.Embed(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	return vectors[0], nil
}

func newTestIndexer(chunks model.DocumentChunksModel, embedder model.Embedder) *Indexer {
	return New(fakeDocuments{}, chunks, embedder, Config{
		Chunk:   model.ChunkConf{Size: 10, Overlap: 2},
		Timeout: time.Second,
	})
}

// 内容未变的分块应复用原向量, 不重复调用向量化
func TestBuildReusesUnchangedChunks(t *testing.T) {
	oldTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	chunks := &fakeChunks{existing: []model.DocumentChunk{{
		DocID:      "doc-1",
		ChunkIndex: 0,
		Text:       "abcdefghij",
		Vector:     []float32{1, 2, 3},
		CreateAt:   oldTime,
	}}}
	embedder := &fakeEmbedder{}

	idx := newTestIndexer(chunks, embedder)
	detail := &model.Documents{Id: "doc-1", OwnerId: "user-1", Content: "abcdefghij"}
	if err := idx.build(context.Background(), detail); err != nil {
		t.Fatalf("build fail: %v", err)
	}

	if len(embedder.embedded) != 0 {
		t.Fatalf("expected no embedding, got %v", embedder.embedded)
	}
	if len(chunks.upserted) != 1 {
		t.Fatalf("expected one chunk, got %d", len(chunks.upserted))
	}
	got := chunks.upserted[0]
	if len(got.Vector) != 3 || got.Vector[0] != 1 {
		t.Fatalf("expected reused vector, got %v", got.Vector)
	}
	if !got.CreateAt.Equal(oldTime) {
		t.Fatalf("expected preserved create_at, got %v", got.CreateAt)
	}
	if len(chunks.deletedAt) != 0 {
		t.Fatalf("expected no tail deletion, got %v", chunks.deletedAt)
	}
}

// 新增内容要向量化, 且带上文档归属与序号
func TestBuildEmbedsNewChunks(t *testing.T) {
	chunks := &fakeChunks{}
	embedder := &fakeEmbedder{}

	idx := newTestIndexer(chunks, embedder)
	detail := &model.Documents{
		Id:      "doc-1",
		OwnerId: "user-1",
		Content: "0123456789abcdefghij", // 20 runes, size 10 overlap 2 -> 3 chunks
	}
	if err := idx.build(context.Background(), detail); err != nil {
		t.Fatalf("build fail: %v", err)
	}

	if len(chunks.upserted) != 3 {
		t.Fatalf("expected 3 chunks, got %d", len(chunks.upserted))
	}
	if len(embedder.embedded) != 3 {
		t.Fatalf("expected 3 embedded texts, got %v", embedder.embedded)
	}
	for i, chunk := range chunks.upserted {
		if chunk.DocID != "doc-1" || chunk.OwnerID != "user-1" {
			t.Fatalf("chunk %d has wrong owner: %+v", i, chunk)
		}
		if chunk.ChunkIndex != int64(i) {
			t.Fatalf("chunk %d has index %d", i, chunk.ChunkIndex)
		}
		if len(chunk.Vector) == 0 {
			t.Fatalf("chunk %d missing vector", i)
		}
		if chunk.CreateAt.IsZero() {
			t.Fatalf("chunk %d missing create_at", i)
		}
	}
}

// 正文变短时删除多余的尾部, 避免过时文本留在索引里
func TestBuildDeletesStaleTail(t *testing.T) {
	chunks := &fakeChunks{existing: []model.DocumentChunk{
		{DocID: "doc-1", ChunkIndex: 0, Text: "abc", Vector: []float32{1}},
		{DocID: "doc-1", ChunkIndex: 1, Text: "def", Vector: []float32{1}},
		{DocID: "doc-1", ChunkIndex: 2, Text: "ghi", Vector: []float32{1}},
	}}
	embedder := &fakeEmbedder{}

	idx := newTestIndexer(chunks, embedder)
	detail := &model.Documents{Id: "doc-1", Content: "短"}
	if err := idx.build(context.Background(), detail); err != nil {
		t.Fatalf("build fail: %v", err)
	}

	if len(chunks.upserted) != 1 {
		t.Fatalf("expected one chunk, got %d", len(chunks.upserted))
	}
	if len(chunks.deletedAt) != 1 || chunks.deletedAt[0] != 1 {
		t.Fatalf("expected tail deletion from 1, got %v", chunks.deletedAt)
	}
}

// 正文清空后应删掉全部分块
func TestBuildEmptyContentClearsChunks(t *testing.T) {
	chunks := &fakeChunks{existing: []model.DocumentChunk{
		{DocID: "doc-1", ChunkIndex: 0, Text: "abc", Vector: []float32{1}},
	}}
	embedder := &fakeEmbedder{}

	idx := newTestIndexer(chunks, embedder)
	detail := &model.Documents{Id: "doc-1", Content: "   "}
	if err := idx.build(context.Background(), detail); err != nil {
		t.Fatalf("build fail: %v", err)
	}

	if len(chunks.upserted) != 0 {
		t.Fatalf("expected no chunk, got %d", len(chunks.upserted))
	}
	if len(chunks.deletedAt) != 1 || chunks.deletedAt[0] != 0 {
		t.Fatalf("expected deletion from 0, got %v", chunks.deletedAt)
	}
	if len(embedder.embedded) != 0 {
		t.Fatalf("expected no embedding, got %v", embedder.embedded)
	}
}

// 文档已删除时应清理历史分块
func TestReindexRemovedDocumentClearsChunks(t *testing.T) {
	chunks := &fakeChunks{}
	idx := New(fakeDocuments{detail: nil, err: model.ErrNotFound}, chunks, &fakeEmbedder{}, Config{})

	idx.reindexDocument("doc-1")

	if !chunks.deletedAll {
		t.Fatal("expected chunks of the removed document to be deleted")
	}
}

func TestScheduleCoalesces(t *testing.T) {
	chunks := &fakeChunks{}
	embedder := &fakeEmbedder{}
	documents := fakeDocuments{detail: &model.Documents{Id: "doc-1", Content: "abcdefghij"}}
	idx := New(documents, chunks, embedder, Config{Chunk: model.ChunkConf{Size: 10, Overlap: 2}})

	idx.Schedule("doc-1")
	idx.Schedule("doc-1")

	waitIdle(t, idx, "doc-1")
	if len(chunks.upserted) == 0 {
		t.Fatal("expected chunks to be upserted")
	}
}

// blockingChunks 首次 Upsert 时阻塞, 用来撑开"重建进行中"的窗口
type blockingChunks struct {
	*fakeChunks
	entered chan struct{}
	release chan struct{}
	once    sync.Once
}

func (b *blockingChunks) Upsert(ctx context.Context, chunks []model.DocumentChunk) error {
	b.once.Do(func() {
		close(b.entered)
		<-b.release
	})
	return b.fakeChunks.Upsert(ctx, chunks)
}

// 重建进行中到来的落库请求不能丢: 靠 dirty 标记再跑一轮, 最终索引最新正文
func TestScheduleDuringRebuildRunsAgain(t *testing.T) {
	chunks := &blockingChunks{
		fakeChunks: &fakeChunks{},
		entered:    make(chan struct{}),
		release:    make(chan struct{}),
	}
	detail := &model.Documents{Id: "doc-1", Content: "AAAA"}
	idx := New(fakeDocuments{detail: detail}, chunks, &fakeEmbedder{}, Config{
		Chunk: model.ChunkConf{Size: 10, Overlap: 2},
	})

	idx.Schedule("doc-1")
	<-chunks.entered

	// 第一轮已读到 AAAA, 此刻正文变成 BBBB 并再次落库
	detail.Content = "BBBB"
	idx.Schedule("doc-1")
	close(chunks.release)

	waitIdle(t, idx, "doc-1")

	if len(chunks.upserted) == 0 {
		t.Fatal("expected chunks to be upserted")
	}
	if got := chunks.upserted[0].Text; got != "BBBB" {
		t.Fatalf("expected the latest content to be indexed, got %q", got)
	}
}

// waitIdle 等待某文档的重建任务结束
func waitIdle(t *testing.T, idx *Indexer, docID string) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		idx.mu.Lock()
		running := idx.running[docID]
		idx.mu.Unlock()
		if !running {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatal("indexer did not finish in time")
}
