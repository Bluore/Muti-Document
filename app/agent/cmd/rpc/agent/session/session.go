package session

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"

	"github.com/Bluore/mutidoc/pkg/ctxdata"
	"github.com/Bluore/mutidoc/pkg/globalkey"
	"github.com/zeromicro/go-zero/core/logx"
)

// ErrSessionClosed 会话已关闭
var ErrSessionClosed = errors.New("session closed")

// Session 会话元信息
type Session struct {
	SessionID    string
	SessionTitle string
	UserID       string
}

// SessionClient 管理内存中单个 agent 会话的生命周期, 内部持有 eino 运行态
type SessionClient struct {
	Session

	// eino 运行态
	Runner       *adk.Runner
	Agent        adk.Agent
	Store        adk.CheckPointStore
	CheckpointID string

	// History 用户界面可见的对话与思考内容
	History *History
	// Memory 大模型实际输入的消息
	Memory []*schema.Message

	ctx       context.Context
	cancel    context.CancelFunc
	createdAt time.Time

	mu     sync.Mutex
	closed bool
	status globalkey.SessionStatus

	conf Config
}

// NewSessionClient 创建会话客户端, 并分配会话级上下文
func NewSessionClient(session Session) *SessionClient {
	ctx, cancel := context.WithCancel(context.Background())
	// 把用户身份写入会话上下文, 供工具做权限校验
	if session.UserID != "" {
		ctx = context.WithValue(ctx, ctxdata.CtxKewJwtUserId, session.UserID)
	}
	return &SessionClient{
		Session:   session,
		History:   NewHistory(),
		Memory:    make([]*schema.Message, 0),
		ctx:       ctx,
		cancel:    cancel,
		createdAt: time.Now(),
	}
}

// GetSession 返回会话元信息
func (c *SessionClient) GetSession() Session {
	return c.Session
}

// Context 返回会话级上下文, 随会话关闭一起取消
func (c *SessionClient) Context() context.Context {
	return c.ctx
}

// SetRuntime 绑定 eino 运行态
func (c *SessionClient) SetRuntime(runner *adk.Runner, agent adk.Agent, store adk.CheckPointStore, checkpointID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Runner = runner
	c.Agent = agent
	c.Store = store
	c.CheckpointID = checkpointID
}

// ResumeCheckpoint 从检查点恢复会话, 返回 eino 事件迭代器
func (c *SessionClient) ResumeCheckpoint(ctx context.Context, params *adk.ResumeParams) (*adk.AsyncIterator[*adk.AgentEvent], error) {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil, ErrSessionClosed
	}
	runner, store, checkpointID := c.Runner, c.Store, c.CheckpointID
	c.mu.Unlock()

	if runner == nil || store == nil {
		logx.Errorf("session %s resume checkpoint fail: runner=%v store=%v",
			c.SessionID, runner != nil, store != nil)
		return nil, ErrNoRunner
	}
	if params == nil {
		params = &adk.ResumeParams{}
	}
	logx.Infof("session %s resume checkpoint: checkpoint_id=%s targets=%d",
		c.SessionID, checkpointID, len(params.Targets))
	return runner.ResumeWithParams(ctx, checkpointID, params)
}

// Close 关闭会话并取消其上下文, 落库后释放 eino 运行态
func (c *SessionClient) Close() {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return
	}
	c.closed = true
	cancel := c.cancel
	c.mu.Unlock()

	if cancel != nil {
		cancel()
	}

	// 先落库(会话历史/记忆/状态), 再释放运行态, 避免持久化与读写锁互相阻塞
	c.persist()

	c.mu.Lock()
	c.Runner = nil
	c.Agent = nil
	c.Store = nil
	c.mu.Unlock()
}

// IsClosed 返回会话是否已关闭
func (c *SessionClient) IsClosed() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.closed
}

// AppendHistory 追加用户界面可见的历史消息
func (c *SessionClient) AppendHistory(msgs ...*schema.Message) {
	c.History.AppendChunks(msgs...)
}

// HistorySSE 返回历史 JSONL 从 offset 开始的增量与新的总长度
func (c *SessionClient) HistorySSE(offset int64) ([]byte, int64) {
	return c.History.ReadFromOffset(offset)
}

// SnapshotHistory 返回用户界面可见的历史消息副本
func (c *SessionClient) SnapshotHistory() []*schema.Message {
	return c.History.Messages()
}

// Runtime 返回绑定的 eino 运行态
func (c *SessionClient) Runtime() (*adk.Runner, string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil, "", ErrSessionClosed
	}
	if c.Runner == nil {
		return nil, "", ErrNoRunner
	}
	return c.Runner, c.CheckpointID, nil
}

// AppendMemory 追加大模型输入消息
func (c *SessionClient) AppendMemory(msgs ...*schema.Message) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Memory = append(c.Memory, msgs...)
}

// SnapshotMemory 返回大模型输入消息副本
func (c *SessionClient) SnapshotMemory() []*schema.Message {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]*schema.Message, len(c.Memory))
	copy(out, c.Memory)
	return out
}

// SetMemory 覆盖大模型输入消息
func (c *SessionClient) SetMemory(msgs []*schema.Message) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Memory = msgs
}

// SetTitle 设置会话标题
func (c *SessionClient) SetTitle(title string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.SessionTitle = title
}

// Status 返回会话运行状态
func (c *SessionClient) Status() globalkey.SessionStatus {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.status
}

// SetStatus 设置会话运行状态
func (c *SessionClient) SetStatus(status globalkey.SessionStatus) {
	c.mu.Lock()
	c.status = status
	c.mu.Unlock()
	c.History.setStatus(status)
}

// TryStartGenerate 尝试进入生成状态, 已在生成中或已关闭时返回 false
func (c *SessionClient) TryStartGenerate() bool {
	c.mu.Lock()
	if c.closed || c.status == globalkey.SessionStatusGenerating {
		c.mu.Unlock()
		return false
	}
	c.status = globalkey.SessionStatusGenerating
	c.mu.Unlock()
	c.History.setStatus(globalkey.SessionStatusGenerating)
	return true
}
