package session

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
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
	History      []*schema.Message

	ctx       context.Context
	cancel    context.CancelFunc
	createdAt time.Time

	mu     sync.Mutex
	closed bool
}

// NewSessionClient 创建会话客户端, 并分配会话级上下文
func NewSessionClient(session Session) *SessionClient {
	ctx, cancel := context.WithCancel(context.Background())
	return &SessionClient{
		Session:   session,
		History:   make([]*schema.Message, 0),
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

// Resume 从检查点恢复会话, 返回 eino 事件迭代器
func (c *SessionClient) Resume(ctx context.Context, params *adk.ResumeParams) (*adk.AsyncIterator[*adk.AgentEvent], error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil, ErrSessionClosed
	}
	// todo: 调用 Runner.ResumeWithParams 续跑未完成的 eino 运行
	return nil, nil
}

// Close 关闭会话并取消其上下文, 释放 eino 运行态
func (c *SessionClient) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return
	}
	c.closed = true
	if c.cancel != nil {
		c.cancel()
	}
	// todo: 持久化 check point 并释放 runner
}

// IsClosed 返回会话是否已关闭
func (c *SessionClient) IsClosed() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.closed
}
