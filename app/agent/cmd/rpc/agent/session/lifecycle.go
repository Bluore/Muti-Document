package session

import (
	"context"
	"errors"

	"github.com/Bluore/mutidoc/app/agent/cmd/rpc/agent/approval"
	"github.com/Bluore/mutidoc/app/agent/model"
	"github.com/Bluore/mutidoc/pkg/globalkey"
	"github.com/Bluore/mutidoc/pkg/idgenerate"

	"github.com/cloudwego/eino/adk"
	"github.com/zeromicro/go-zero/core/logx"
)

// 会话标题最大长度
const sessionTitleMaxRunes = 30

var (
	// ErrGenerating 会话正在生成中
	ErrGenerating = errors.New("session is generating")
	// ErrNoRunner 未绑定 eino 运行态
	ErrNoRunner = errors.New("session runner is not ready")
	// ErrInterrupted 会话被中断, 等待人反馈
	ErrInterrupted = errors.New("session interrupted")
	// ErrNotInterrupted 会话当前没有待反馈的打断点
	ErrNotInterrupted = errors.New("session is not interrupted")
)

// Config 会话客户端构造依赖
type Config struct {
	Model           model.AgentSessionsModel
	Agent           adk.Agent
	CheckpointStore adk.CheckPointStore
	ServerID        int64
}

// Create 创建会话, 由会话客户端负责落库
func Create(ctx context.Context, conf Config, userID, title string) (*SessionClient, error) {
	client := NewSessionClient(Session{
		SessionID:    idgenerate.NewID(conf.ServerID),
		SessionTitle: title,
		UserID:       userID,
	})
	client.conf = conf
	client.bind()

	row := &model.AgentSessions{
		Id:      client.SessionID,
		UserId:  client.UserID,
		Title:   client.SessionTitle,
		History: "",
		Memory:  "",
		Status:  int64(globalkey.SessionStatusIdle),
		Version: 1,
	}
	if _, err := conf.Model.Insert(ctx, nil, row); err != nil {
		return nil, err
	}
	if err := Store(client); err != nil {
		return nil, err
	}
	return client, nil
}

// Get 读取会话, 内存中不存在时由客户端自动读库重建
func Get(ctx context.Context, conf Config, sessionID string) (*SessionClient, error) {
	return GetOrInit(sessionID, func() (*SessionClient, error) {
		row, err := conf.Model.FindOne(ctx, sessionID)
		if err != nil {
			return nil, err
		}
		return fromRow(conf, row)
	})
}

// Generate 提交一轮问答: 标记生成中并在后台执行, 完成后落库
func (c *SessionClient) Generate(query string) error {
	if c == nil {
		return ErrNilSession
	}
	if c.isClosed() {
		return ErrSessionClosed
	}
	// 有未处理的打断时必须先完成人工反馈, 避免覆盖检查点
	if c.Status() == globalkey.SessionStatusInterrupted {
		return ErrInterrupted
	}

	return c.startTurn(func(ctx context.Context) error {
		// 首轮用提问生成标题
		if c.GetSession().SessionTitle == "" {
			c.setTitle(truncateRunes(query, sessionTitleMaxRunes))
		}
		return c.generateTurn(ctx, query)
	})
}

// HumanFeedback 提交人工审批结果并恢复被打断的执行
func (c *SessionClient) HumanFeedback(approved bool, feedback string) error {
	if c == nil {
		return ErrNilSession
	}
	if c.isClosed() {
		return ErrSessionClosed
	}
	if c.Status() == globalkey.SessionStatusGenerating {
		return ErrGenerating
	}
	if c.Status() != globalkey.SessionStatusInterrupted || !c.hasPendingInterrupt() {
		return ErrNotInterrupted
	}

	decision := approval.Decision{Approved: approved, Feedback: feedback}
	return c.startTurn(func(ctx context.Context) error {
		return c.resumeTurn(ctx, decision)
	})
}

// startTurn 标记生成中并在后台执行一轮, 结束后更新状态并落库
func (c *SessionClient) startTurn(turn func(context.Context) error) error {
	if !c.tryStartGenerate() {
		return ErrGenerating
	}

	go func() {
		status := globalkey.SessionStatusDone
		logx.Infof("session %s turn start", c.SessionID)
		switch err := turn(c.context()); {
		case errors.Is(err, ErrInterrupted):
			status = globalkey.SessionStatusInterrupted
			logx.Errorf("session %s turn interrupted, waiting for human feedback", c.SessionID)
		case err != nil:
			status = globalkey.SessionStatusFailed
			logx.Errorf("session %s turn fail: %v", c.SessionID, err)
		}
		logx.Infof("session %s turn end, status=%d", c.SessionID, status)
		c.setStatus(status)
		c.persist()
	}()
	return nil
}

// bind 绑定 eino 运行态
func (c *SessionClient) bind() {
	runner := adk.NewRunner(context.Background(), adk.RunnerConfig{
		Agent:           c.conf.Agent,
		EnableStreaming: true,
		CheckPointStore: c.conf.CheckpointStore,
	})
	c.setRuntime(runner, c.conf.Agent, c.conf.CheckpointStore, c.SessionID)
}

// fromRow 由数据库记录重建会话客户端; 库列 history/memory 分别对应输出/上下文
func fromRow(conf Config, row *model.AgentSessions) (*SessionClient, error) {
	msgs, err := DecodeMessages([]byte(row.Memory))
	if err != nil {
		return nil, err
	}

	client := NewSessionClient(Session{
		SessionID:    row.Id,
		SessionTitle: row.Title,
		UserID:       row.UserId,
	})
	client.conf = conf
	if err = client.Output.Load([]byte(row.History)); err != nil {
		return nil, err
	}
	client.appendContext(msgs...)

	status := globalkey.SessionStatus(row.Status)
	// 待反馈的打断点只保存在内存, 重启后无法恢复, 降级为失败避免会话卡死
	if status == globalkey.SessionStatusInterrupted {
		status = globalkey.SessionStatusFailed
	}
	client.setStatus(status)
	client.bind()
	return client, nil
}

// persist 把会话输出/上下文/状态写回数据库
func (c *SessionClient) persist() {
	if c.conf.Model == nil {
		return
	}
	ctx := context.Background()

	msgs, err := EncodeMessages(c.snapshotContext())
	if err != nil {
		logx.Errorf("encode session context fail: %v", err)
		return
	}

	row, err := c.conf.Model.FindOne(ctx, c.SessionID)
	if err != nil {
		logx.Errorf("find session %s fail: %v", c.SessionID, err)
		return
	}

	meta := c.GetSession()
	row.Title = meta.SessionTitle
	row.History = string(c.Output.Bytes())
	row.Memory = string(msgs)
	row.Status = int64(c.Status())
	if err = c.conf.Model.UpdateWithVersion(ctx, nil, row); err != nil {
		logx.Errorf("update session %s fail: %v", c.SessionID, err)
	}
}

// truncateRunes 按字符截断
func truncateRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}
