package session

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"

	"github.com/Bluore/mutidoc/app/agent/cmd/rpc/agent/botconn"
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

	// Output 用户界面可见的对话与思考内容, 只追加
	Output *Output
	// Context 大模型实际输入的消息, 只追加
	Context []*schema.Message

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
	// 挂载机器人连接追踪器, 供 EditAgent 结束后统一退出
	ctx = botconn.WithTracker(ctx)
	return &SessionClient{
		Session:   session,
		Output:    NewOutput(),
		Context:   make([]*schema.Message, 0),
		ctx:       ctx,
		cancel:    cancel,
		createdAt: time.Now(),
	}
}

// GetSession 返回会话元信息
func (c *SessionClient) GetSession() Session {
	return c.Session
}

// context 返回会话级上下文, 随会话关闭一起取消
func (c *SessionClient) context() context.Context {
	return c.ctx
}

// setRuntime 绑定 eino 运行态
func (c *SessionClient) setRuntime(runner *adk.Runner, agent adk.Agent, store adk.CheckPointStore, checkpointID string) {
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

	// 先落库(输出/上下文/状态), 再释放运行态, 避免持久化与读写锁互相阻塞
	c.persist()

	c.mu.Lock()
	c.Runner = nil
	c.Agent = nil
	c.Store = nil
	c.mu.Unlock()
}

// isClosed 返回会话是否已关闭
func (c *SessionClient) isClosed() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.closed
}

// appendOutput 追加用户界面可见的输出消息
func (c *SessionClient) appendOutput(msgs ...*schema.Message) {
	c.Output.AppendChunks(msgs...)
}

// OutputSSE 返回输出 JSONL 从 offset 开始的增量与新的总长度
func (c *SessionClient) OutputSSE(offset int64) ([]byte, int64) {
	return c.Output.ReadFromOffset(offset)
}

// snapshotOutput 返回用户界面可见的输出消息副本
func (c *SessionClient) snapshotOutput() []*schema.Message {
	return c.Output.Messages()
}

// runtime 返回绑定的 eino 运行态
func (c *SessionClient) runtime() (*adk.Runner, string, error) {
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

// appendContext 追加大模型输入消息
func (c *SessionClient) appendContext(msgs ...*schema.Message) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Context = append(c.Context, msgs...)
}

// snapshotContext 返回大模型输入消息副本
func (c *SessionClient) snapshotContext() []*schema.Message {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]*schema.Message, len(c.Context))
	copy(out, c.Context)
	return out
}

// setTitle 设置会话标题
func (c *SessionClient) setTitle(title string) {
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

// setStatus 设置会话运行状态
func (c *SessionClient) setStatus(status globalkey.SessionStatus) {
	c.mu.Lock()
	c.status = status
	c.mu.Unlock()
	c.Output.setStatus(status)
}

// tryStartGenerate 尝试进入生成状态, 已在生成中或已关闭时返回 false
func (c *SessionClient) tryStartGenerate() bool {
	c.mu.Lock()
	if c.closed || c.status == globalkey.SessionStatusGenerating {
		c.mu.Unlock()
		return false
	}
	c.status = globalkey.SessionStatusGenerating
	c.mu.Unlock()
	c.Output.setStatus(globalkey.SessionStatusGenerating)
	return true
}
