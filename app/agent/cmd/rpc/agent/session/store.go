package session

import (
	"fmt"
	"sync"
)

// 全局内存会话表
var (
	sessionsMu sync.RWMutex
	sessions   = make(map[string]*SessionClient)
)

// Store 写入全局内存会话表
func Store(client *SessionClient) error {
	if client == nil {
		return fmt.Errorf("session client is nil")
	}
	sessionsMu.Lock()
	defer sessionsMu.Unlock()
	if _, ok := sessions[client.SessionID]; ok {
		return fmt.Errorf("session already exists")
	}
	sessions[client.SessionID] = client
	return nil
}

// getClient 从全局内存会话表读取会话
func getClient(sessionID string) (*SessionClient, bool) {
	sessionsMu.RLock()
	defer sessionsMu.RUnlock()
	client, ok := sessions[sessionID]
	return client, ok
}

// GetOrInit 读取内存会话, 不存在时用 build 构造并写入
func GetOrInit(sessionID string, build func() (*SessionClient, error)) (*SessionClient, error) {
	if client, ok := getClient(sessionID); ok {
		return client, nil
	}

	client, err := build()
	if err != nil {
		return nil, err
	}
	if err = Store(client); err != nil {
		// 并发下可能已被其他请求初始化
		if existing, ok := getClient(sessionID); ok {
			return existing, nil
		}
		return nil, err
	}
	return client, nil
}

// Delete 从全局内存会话表移除会话, 并关闭其生命周期
func Delete(sessionID string) {
	sessionsMu.Lock()
	client, ok := sessions[sessionID]
	if ok {
		delete(sessions, sessionID)
	}
	sessionsMu.Unlock()

	if ok {
		client.Close()
	}
}

// ListByUser 返回某用户的全部会话元信息
func ListByUser(userID string) []Session {
	sessionsMu.RLock()
	defer sessionsMu.RUnlock()

	list := make([]Session, 0, len(sessions))
	for _, client := range sessions {
		if client.UserID == userID {
			list = append(list, client.GetSession())
		}
	}
	return list
}

// Len 返回当前内存中的会话数
func Len() int {
	sessionsMu.RLock()
	defer sessionsMu.RUnlock()
	return len(sessions)
}
