package session

import "sync"

// 全局内存会话表
var (
	sessionsMu sync.RWMutex
	sessions   = make(map[string]*SessionClient)
)

// Store 写入全局内存会话表
func Store(client *SessionClient) {
	if client == nil {
		return
	}
	sessionsMu.Lock()
	defer sessionsMu.Unlock()
	sessions[client.SessionID] = client
}

// Get 从全局内存会话表读取会话
func Get(sessionID string) (*SessionClient, bool) {
	sessionsMu.RLock()
	defer sessionsMu.RUnlock()
	client, ok := sessions[sessionID]
	return client, ok
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
