// Package approval 管理敏感操作的人工审批信息与本次会话的授权白名单。
package approval

import (
	"context"
	"fmt"
	"sync"

	"github.com/cloudwego/eino/schema"
)

// 打断类型
const KindCreateDocumentConnection = "create_document_connection"

// Info 敏感操作的人工确认信息, 随打断事件推送给前端
type Info struct {
	Kind       string `json:"kind"`
	DocumentID string `json:"document_id"`
	BotName    string `json:"bot_name"`
	Message    string `json:"message"`
}

// Decision 用户对敏感操作的审批结果, 恢复执行时回传给工具
type Decision struct {
	Approved bool   `json:"approved"`
	Feedback string `json:"feedback,omitempty"`
}

// 打断信息会随 eino 检查点序列化, 需要注册具体类型
func init() {
	schema.Register[Info]()
	schema.Register[Decision]()
}

// NewInfo 构造建立文档编辑连接的人工确认信息
func NewInfo(documentID, botName string) Info {
	return Info{
		Kind:       KindCreateDocumentConnection,
		DocumentID: documentID,
		BotName:    botName,
		Message: fmt.Sprintf("智能助手需要建立对文档「%s」的编辑连接（机器人：%s），获得授权后可能修改文档内容。是否同意？",
			documentID, botName),
	}
}

type trackerKey struct{}

// Tracker 记录本次会话已授权的文档
type Tracker struct {
	mu  sync.Mutex
	ids map[string]struct{}
}

// WithTracker 在 ctx 上挂载审批白名单, 已存在时复用
func WithTracker(ctx context.Context) context.Context {
	if _, ok := ctx.Value(trackerKey{}).(*Tracker); ok {
		return ctx
	}
	return context.WithValue(ctx, trackerKey{}, &Tracker{ids: make(map[string]struct{})})
}

// Allow 把文档加入本次会话的授权白名单; ctx 上没有追踪器时忽略
func Allow(ctx context.Context, documentID string) {
	if documentID == "" {
		return
	}
	t, ok := ctx.Value(trackerKey{}).(*Tracker)
	if !ok {
		return
	}
	t.mu.Lock()
	t.ids[documentID] = struct{}{}
	t.mu.Unlock()
}

// Allowed 返回文档是否已在本次会话中授权
func Allowed(ctx context.Context, documentID string) bool {
	if documentID == "" {
		return false
	}
	t, ok := ctx.Value(trackerKey{}).(*Tracker)
	if !ok {
		return false
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	_, ok = t.ids[documentID]
	return ok
}
