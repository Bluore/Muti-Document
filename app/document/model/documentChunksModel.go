package model

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/qdrant/go-client/qdrant"
	"github.com/zeromicro/go-zero/core/logx"
)

// 向量名与 payload 字段名, 与 deploy/qdrant/document_chunks.json 保持一致。
const (
	// DocumentChunkVectorName 是 document_chunks 的向量字段名
	DocumentChunkVectorName = "chunk"

	documentChunkFieldDocID    = "doc_id"
	documentChunkFieldOwnerID  = "owner_id"
	documentChunkFieldIndex    = "chunk_index"
	documentChunkFieldText     = "text"
	documentChunkFieldCreateAt = "create_at"

	defaultChunkCollection = "document_chunks"
	// defaultChunkVectorSize 默认向量维度, 与 BAAI/bge-m3 一致
	defaultChunkVectorSize = 1024
	// defaultSearchTopK 检索未指定条数时返回的分块数
	defaultSearchTopK = 3
	// chunkUpsertBatch 单次写入的点数上限
	chunkUpsertBatch = 64
)

var _ DocumentChunksModel = (*defaultDocumentChunksModel)(nil)

// QdrantConf 文档向量库连接配置
type QdrantConf struct {
	Host       string
	Port       int
	APIKey     string
	UseTLS     bool
	Collection string
	VectorSize int
}

// DocumentChunksModel 是文档分块向量表(document_chunks 集合)的读写操作
type DocumentChunksModel interface {
	// DeleteByDocID 删除文档的全部历史分块, 用于清理过时文本
	DeleteByDocID(ctx context.Context, docID string) error
	// DeleteFrom 删除文档序号 >= fromIndex 的分块, 用于清理正文变短后的过时尾部
	DeleteFrom(ctx context.Context, docID string, fromIndex int64) error
	// FindByDocID 读取文档现有分块(含向量), 用于重建索引时复用未变内容
	FindByDocID(ctx context.Context, docID string) ([]DocumentChunk, error)
	// Upsert 写入分块与向量
	Upsert(ctx context.Context, chunks []DocumentChunk) error
	// Search 按向量检索相似分块, 返回相似度最高的 topK 条
	Search(ctx context.Context, vector []float32, topK int, filter DocumentChunkFilter) ([]DocumentChunk, error)
}

type defaultDocumentChunksModel struct {
	client     *qdrant.Client
	collection string
	vectorSize uint64

	// ensureMu 保护集合与索引的懒创建, 避免每次读写都探测集合是否存在
	ensureMu sync.Mutex
	ensured  bool
}

// NewDocumentChunksModel 连接 Qdrant 并返回文档分块模型。
// 集合与 payload 索引在首次读写时幂等创建, 结果与 deploy/qdrant/init.sh 一致。
// 连接按需建立, 向量库不可用时不会拖慢服务启动, 读写时才报错。
func NewDocumentChunksModel(c QdrantConf) (DocumentChunksModel, error) {
	collection := c.Collection
	if collection == "" {
		collection = defaultChunkCollection
	}
	vectorSize := c.VectorSize
	if vectorSize <= 0 {
		vectorSize = defaultChunkVectorSize
	}

	client, err := qdrant.NewClient(&qdrant.Config{
		Host:   c.Host,
		Port:   c.Port,
		APIKey: c.APIKey,
		UseTLS: c.UseTLS,
		// 连接按需建立, 跳过启动时的版本探测, 避免向量库不可用时拖慢服务启动
		SkipCompatibilityCheck: true,
		PoolSize:               1,
	})
	if err != nil {
		return nil, fmt.Errorf("connect qdrant: %w", err)
	}

	return &defaultDocumentChunksModel{
		client:     client,
		collection: collection,
		vectorSize: uint64(vectorSize),
	}, nil
}

// ensure 幂等创建集合与索引; 失败不记状态, 下次读写会重试
func (m *defaultDocumentChunksModel) ensure(ctx context.Context) error {
	m.ensureMu.Lock()
	defer m.ensureMu.Unlock()
	if m.ensured {
		return nil
	}
	if err := m.initCollection(ctx); err != nil {
		return err
	}
	m.ensured = true
	return nil
}

func (m *defaultDocumentChunksModel) initCollection(ctx context.Context) error {
	exists, err := m.client.CollectionExists(ctx, m.collection)
	if err != nil {
		return fmt.Errorf("check collection %s: %w", m.collection, err)
	}
	if exists {
		return nil
	}

	err = m.client.CreateCollection(ctx, &qdrant.CreateCollection{
		CollectionName: m.collection,
		VectorsConfig: qdrant.NewVectorsConfigMap(map[string]*qdrant.VectorParams{
			DocumentChunkVectorName: {
				Size:     m.vectorSize,
				Distance: qdrant.Distance_Cosine,
			},
		}),
		HnswConfig: &qdrant.HnswConfigDiff{
			M:           qdrant.PtrOf(uint64(16)),
			EfConstruct: qdrant.PtrOf(uint64(100)),
		},
		OptimizersConfig: &qdrant.OptimizersConfigDiff{
			DefaultSegmentNumber: qdrant.PtrOf(uint64(2)),
			IndexingThreshold:    qdrant.PtrOf(uint64(20000)),
		},
		WalConfig: &qdrant.WalConfigDiff{
			WalCapacityMb: qdrant.PtrOf(uint64(32)),
		},
	})
	if err != nil {
		return fmt.Errorf("create collection %s: %w", m.collection, err)
	}
	logx.Infow("create qdrant collection",
		logx.Field("collection", m.collection),
		logx.Field("vector_size", m.vectorSize),
	)

	indexes := []*qdrant.CreateFieldIndexCollection{
		{
			FieldName: documentChunkFieldDocID,
			FieldType: qdrant.PtrOf(qdrant.FieldType_FieldTypeKeyword),
		},
		{
			FieldName: documentChunkFieldOwnerID,
			FieldType: qdrant.PtrOf(qdrant.FieldType_FieldTypeKeyword),
		},
		{
			FieldName: documentChunkFieldIndex,
			FieldType: qdrant.PtrOf(qdrant.FieldType_FieldTypeInteger),
		},
		{
			FieldName: documentChunkFieldCreateAt,
			FieldType: qdrant.PtrOf(qdrant.FieldType_FieldTypeDatetime),
		},
		{
			FieldName: documentChunkFieldText,
			FieldType: qdrant.PtrOf(qdrant.FieldType_FieldTypeText),
			FieldIndexParams: qdrant.NewPayloadIndexParamsText(&qdrant.TextIndexParams{
				Tokenizer:   qdrant.TokenizerType_Multilingual,
				Lowercase:   qdrant.PtrOf(true),
				MinTokenLen: qdrant.PtrOf(uint64(2)),
			}),
		},
	}
	for _, index := range indexes {
		index.CollectionName = m.collection
		index.Wait = qdrant.PtrOf(true)
		// 索引只影响查询性能, 失败不阻断读写
		if _, err := m.client.CreateFieldIndex(ctx, index); err != nil {
			logx.Errorw("create qdrant payload index fail",
				logx.Field("error", err),
				logx.Field("collection", m.collection),
				logx.Field("field", index.FieldName),
			)
		}
	}
	return nil
}

// DeleteByDocID 删除文档的全部历史分块
func (m *defaultDocumentChunksModel) DeleteByDocID(ctx context.Context, docID string) error {
	return m.deleteByFilter(ctx, docID, nil)
}

// DeleteFrom 删除文档序号 >= fromIndex 的分块
func (m *defaultDocumentChunksModel) DeleteFrom(ctx context.Context, docID string, fromIndex int64) error {
	if fromIndex < 0 {
		return errors.New("document chunk: negative chunk index")
	}
	return m.deleteByFilter(ctx, docID, []*qdrant.Condition{
		qdrant.NewRange(documentChunkFieldIndex, &qdrant.Range{Gte: qdrant.PtrOf(float64(fromIndex))}),
	})
}

func (m *defaultDocumentChunksModel) deleteByFilter(
	ctx context.Context,
	docID string,
	mustNot []*qdrant.Condition,
) error {
	if docID == "" {
		return errors.New("document chunk: empty document id")
	}
	if err := m.ensure(ctx); err != nil {
		return err
	}

	_, err := m.client.Delete(ctx, &qdrant.DeletePoints{
		CollectionName: m.collection,
		Wait:           qdrant.PtrOf(true),
		Points: qdrant.NewPointsSelectorFilter(&qdrant.Filter{
			Must:    []*qdrant.Condition{qdrant.NewMatchKeyword(documentChunkFieldDocID, docID)},
			MustNot: mustNot,
		}),
	})
	if err != nil {
		return fmt.Errorf("delete chunks of %s: %w", docID, err)
	}
	return nil
}

// FindByDocID 读取文档现有分块(含向量)
func (m *defaultDocumentChunksModel) FindByDocID(ctx context.Context, docID string) ([]DocumentChunk, error) {
	if docID == "" {
		return nil, errors.New("document chunk: empty document id")
	}
	if err := m.ensure(ctx); err != nil {
		return nil, err
	}

	iter := m.client.ScrollAll(ctx, &qdrant.ScrollPoints{
		CollectionName: m.collection,
		Filter:         chunkFilter(DocumentChunkFilter{DocID: docID}),
		WithPayload:    qdrant.NewWithPayload(true),
		WithVectors:    qdrant.NewWithVectors(true),
	})

	var chunks []DocumentChunk
	for {
		points, err := iter.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("scroll chunks of %s: %w", docID, err)
		}
		for _, point := range points {
			chunks = append(chunks, pointToChunk(point))
		}
	}
	return chunks, nil
}

// Upsert 分批写入分块与向量
func (m *defaultDocumentChunksModel) Upsert(ctx context.Context, chunks []DocumentChunk) error {
	if len(chunks) == 0 {
		return nil
	}
	if err := m.ensure(ctx); err != nil {
		return err
	}

	for start := 0; start < len(chunks); start += chunkUpsertBatch {
		end := start + chunkUpsertBatch
		if end > len(chunks) {
			end = len(chunks)
		}

		points := make([]*qdrant.PointStruct, 0, end-start)
		for _, chunk := range chunks[start:end] {
			if len(chunk.Vector) == 0 {
				return fmt.Errorf("document chunk %s#%d: empty vector", chunk.DocID, chunk.ChunkIndex)
			}
			points = append(points, chunkToPoint(chunk))
		}

		if _, err := m.client.Upsert(ctx, &qdrant.UpsertPoints{
			CollectionName: m.collection,
			Wait:           qdrant.PtrOf(true),
			Points:         points,
		}); err != nil {
			return fmt.Errorf("upsert chunks: %w", err)
		}
	}
	return nil
}

// Search 按向量检索相似分块
func (m *defaultDocumentChunksModel) Search(
	ctx context.Context,
	vector []float32,
	topK int,
	filter DocumentChunkFilter,
) ([]DocumentChunk, error) {
	if len(vector) == 0 {
		return nil, errors.New("document chunk: empty query vector")
	}
	if topK <= 0 {
		topK = defaultSearchTopK
	}
	if err := m.ensure(ctx); err != nil {
		return nil, err
	}

	points, err := m.client.Query(ctx, &qdrant.QueryPoints{
		CollectionName: m.collection,
		Query:          qdrant.NewQueryDense(vector),
		Using:          qdrant.PtrOf(DocumentChunkVectorName),
		Filter:         chunkFilter(filter),
		Limit:          qdrant.PtrOf(uint64(topK)),
		WithPayload:    qdrant.NewWithPayload(true),
	})
	if err != nil {
		return nil, fmt.Errorf("search chunks: %w", err)
	}

	chunks := make([]DocumentChunk, 0, len(points))
	for _, point := range points {
		chunk := payloadToChunk(point.Payload)
		chunk.Score = point.Score
		chunks = append(chunks, chunk)
	}
	return chunks, nil
}

// chunkFilter 把过滤条件翻译成 Qdrant filter, 无条件下返回 nil
func chunkFilter(f DocumentChunkFilter) *qdrant.Filter {
	must := make([]*qdrant.Condition, 0, 2)
	if f.DocID != "" {
		must = append(must, qdrant.NewMatchKeyword(documentChunkFieldDocID, f.DocID))
	}
	if f.OwnerID != "" {
		must = append(must, qdrant.NewMatchKeyword(documentChunkFieldOwnerID, f.OwnerID))
	}
	if len(must) == 0 {
		return nil
	}
	return &qdrant.Filter{Must: must}
}

func chunkToPoint(chunk DocumentChunk) *qdrant.PointStruct {
	return &qdrant.PointStruct{
		Id: qdrant.NewID(documentChunkPointID(chunk.DocID, chunk.ChunkIndex)),
		Vectors: qdrant.NewVectorsMap(map[string]*qdrant.Vector{
			DocumentChunkVectorName: qdrant.NewVectorDense(chunk.Vector),
		}),
		Payload: qdrant.NewValueMap(map[string]any{
			documentChunkFieldDocID:    chunk.DocID,
			documentChunkFieldOwnerID:  chunk.OwnerID,
			documentChunkFieldIndex:    chunk.ChunkIndex,
			documentChunkFieldText:     chunk.Text,
			documentChunkFieldCreateAt: chunk.CreateAt.UTC().Format(time.RFC3339),
		}),
	}
}

func pointToChunk(point *qdrant.RetrievedPoint) DocumentChunk {
	chunk := payloadToChunk(point.Payload)
	if point.Vectors == nil {
		return chunk
	}
	named := point.Vectors.GetVectors()
	if named == nil {
		return chunk
	}
	if vector, ok := named.GetVectors()[DocumentChunkVectorName]; ok {
		if dense := vector.GetDenseVector(); dense != nil {
			chunk.Vector = dense.GetData()
		}
	}
	return chunk
}

func payloadToChunk(payload map[string]*qdrant.Value) DocumentChunk {
	return DocumentChunk{
		DocID:      payload[documentChunkFieldDocID].GetStringValue(),
		OwnerID:    payload[documentChunkFieldOwnerID].GetStringValue(),
		ChunkIndex: payload[documentChunkFieldIndex].GetIntegerValue(),
		Text:       payload[documentChunkFieldText].GetStringValue(),
		CreateAt:   parseChunkCreateAt(payload[documentChunkFieldCreateAt].GetStringValue()),
	}
}

func parseChunkCreateAt(raw string) time.Time {
	createAt, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}
	}
	return createAt
}
