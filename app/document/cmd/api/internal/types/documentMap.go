package types

import (
	"sync"

	"github.com/Bluore/mutidoc/app/document/cmd/api/internal/connection"
)

type DocumentMap struct {
	docs map[string]*connection.DocumentClient
	mu   *sync.RWMutex
}

func NewDocumentMap() *DocumentMap {
	return &DocumentMap{
		docs: make(map[string]*connection.DocumentClient),
	}
}

func (m DocumentMap) Get(key string) *connection.DocumentClient {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.docs[key]
}

func (m DocumentMap) GetOrCreate(key string) *connection.DocumentClient {
	m.mu.Lock()
	defer m.mu.Unlock()
	if doc, ok := m.docs[key]; ok {
		return doc
	}

	document := connection.NewOt()
	m.docs[key] = document
	return document
}

func (m DocumentMap) Set(key string, value *connection.DocumentClient) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.docs[key] = value
}

func (m DocumentMap) CheckAndDelete(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.docs[key]; ok {
		delete(m.docs, key)
	}
}

func (m DocumentMap) Len() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.docs)
}
