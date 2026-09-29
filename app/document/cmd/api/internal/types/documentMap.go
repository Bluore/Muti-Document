package types

import (
	"sync"

	"github.com/Bluore/mutidoc/app/document/cmd/api/internal/connection"
	"github.com/Bluore/mutidoc/app/document/model"
	"github.com/redis/go-redis/v9"
)

type DocumentMap struct {
	docs map[string]*connection.DocumentClient
	mu   *sync.RWMutex

	redisClient   *redis.Client
	documentModel model.DocumentsModel
}

func NewDocumentMap(redisClient *redis.Client, documentModel model.DocumentsModel) *DocumentMap {
	return &DocumentMap{
		docs:          make(map[string]*connection.DocumentClient, 100),
		mu:            &sync.RWMutex{},
		redisClient:   redisClient,
		documentModel: documentModel,
	}
}

func (m DocumentMap) Get(key string) *connection.DocumentClient {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.docs[key]
}

// GetOrCreate 返回文档会话; 不存在或已关闭时用数据库快照重新创建。
// 创建时会合并 redis 上未落库的变更。
func (m DocumentMap) GetOrCreate(key string, detail model.Documents) *connection.DocumentClient {
	m.mu.Lock()
	defer m.mu.Unlock()
	if doc, ok := m.docs[key]; ok && !doc.IsClosed() {
		return doc
	}

	document := connection.NewOt(detail, m.redisClient, m.documentModel, func(client *connection.DocumentClient) {
		m.CheckAndDelete(key, client)
	})
	m.docs[key] = document
	return document
}

func (m DocumentMap) Set(key string, value *connection.DocumentClient) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.docs[key] = value
}

// CheckAndDelete 仅在当前会话仍是目标会话时删除, 避免误删重新创建的新会话
func (m DocumentMap) CheckAndDelete(key string, target *connection.DocumentClient) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if current, ok := m.docs[key]; ok && current == target {
		delete(m.docs, key)
	}
}

func (m DocumentMap) Len() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.docs)
}
