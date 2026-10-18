package session

import (
	"context"
	"errors"

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
	if !c.tryStartGenerate() {
		return ErrGenerating
	}

	// 首轮用提问生成标题
	if c.GetSession().SessionTitle == "" {
		c.setTitle(truncateRunes(query, sessionTitleMaxRunes))
	}

	go func() {
		status := globalkey.SessionStatusDone
		logx.Infof("session %s generate start", c.SessionID)
		switch err := c.generateTurn(c.context(), query); {
		case errors.Is(err, ErrInterrupted):
			status = globalkey.SessionStatusInterrupted
			logx.Errorf("session %s generate interrupted, waiting for human feedback", c.SessionID)
		case err != nil:
			status = globalkey.SessionStatusFailed
			logx.Errorf("session %s generate fail: %v", c.SessionID, err)
		}
		logx.Infof("session %s generate end, status=%d", c.SessionID, status)
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
	client.setStatus(globalkey.SessionStatus(row.Status))
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
