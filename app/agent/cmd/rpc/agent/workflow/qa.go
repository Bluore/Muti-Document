package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/Bluore/mutidoc/app/agent/cmd/rpc/agent/session"
	"github.com/Bluore/mutidoc/app/agent/cmd/rpc/protocol"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
	"github.com/zeromicro/go-zero/core/logx"
)

// 一次问答工作流错误
var (
	ErrNilSession = errors.New("session client is nil")
	ErrNoRunner   = errors.New("session runner is not ready")
	ErrNoAnswer   = errors.New("agent produced no answer")
)

// QA 一次问答工作流: 驱动会话跑完一轮, 收集助手回复并写回会话历史
type QA struct {
	client *session.SessionClient
}

// NewQA 基于会话客户端创建一次问答工作流
func NewQA(client *session.SessionClient) *QA {
	return &QA{client: client}
}

// Run 执行一次问答, 作为 session.RunFunc 注入
func Run(ctx context.Context, client *session.SessionClient, query string) error {
	return NewQA(client).Ask(ctx, query)
}

// Ask 进行一次问答: 逐 chunk 写入可见历史(打字机效果), 结束后用完整消息重建模型记忆
func (w *QA) Ask(ctx context.Context, query string) error {
	if w.client == nil {
		return ErrNilSession
	}
	if w.client.IsClosed() {
		return session.ErrSessionClosed
	}

	logx.WithContext(ctx).Infow("qa: turn start",
		logx.Field("session_id", w.client.SessionID),
		logx.Field("query_len", len(query)),
	)

	// 用户消息写入可见历史, 同时作为本轮模型输入
	w.client.AppendHistory(schema.UserMessage(query))
	w.client.AppendMemory(schema.UserMessage(query))

	iter, err := w.Resume(ctx, query)
	if err != nil {
		logx.WithContext(ctx).Errorf("qa: run fail | session=%s err=%v", w.client.SessionID, err)
		return err
	}

	answered := false
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			logx.WithContext(ctx).Errorf("qa: agent event error | session=%s agent=%s run_path=%s err=%v",
				w.client.SessionID, event.AgentName, formatRunPath(event.RunPath), event.Err)
			return event.Err
		}
		if event.Action != nil && event.Action.Interrupted != nil {
			w.logInterrupt(ctx, event)
			w.recordInterrupt(event.Action.Interrupted)
			return session.ErrInterrupted
		}
		if event.Output == nil || event.Output.MessageOutput == nil {
			continue
		}

		mo := event.Output.MessageOutput
		if mo.IsStreaming && mo.MessageStream != nil {
			// 流式: 每个分片即时写入, SSE 轮询即可增量返回
			if err = w.stream(mo.MessageStream, mo.Role, &answered); err != nil {
				logx.WithContext(ctx).Errorf("qa: consume stream fail | session=%s agent=%s err=%v",
					w.client.SessionID, event.AgentName, err)
				return err
			}
			w.client.History.FinalizeMessage()
			continue
		}
		if mo.Message != nil {
			w.appendVisible(mo.Message, &answered)
			w.client.History.FinalizeMessage()
		}
	}

	if !answered {
		logx.WithContext(ctx).Errorf("qa: no answer produced | session=%s", w.client.SessionID)
		return ErrNoAnswer
	}

	// 用完整消息(去思考)重建模型记忆, 供下一轮输入与落库
	w.client.SetMemory(withoutReasoningAll(w.client.SnapshotHistory()))
	logx.WithContext(ctx).Infow("qa: turn done",
		logx.Field("session_id", w.client.SessionID),
		logx.Field("history_len", len(w.client.SnapshotHistory())),
	)
	return nil
}

// Resume 以当前记忆作为输入驱动一轮生成
func (w *QA) Resume(ctx context.Context, query string) (*adk.AsyncIterator[*adk.AgentEvent], error) {
	if w.client == nil {
		return nil, ErrNilSession
	}
	runner, checkpointID, err := w.client.Runtime()
	if err != nil {
		return nil, err
	}
	return runner.Run(ctx, w.client.SnapshotMemory(), adk.WithCheckPointID(checkpointID)), nil
}

// stream 逐分片消费一条流式消息
func (w *QA) stream(stream *schema.StreamReader[*schema.Message], role schema.RoleType, answered *bool) error {
	for {
		chunk, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		if chunk == nil {
			continue
		}
		if chunk.Role == "" {
			chunk.Role = role
		}
		w.appendVisible(chunk, answered)
	}
}

// appendVisible 追加上一条可见消息, 非用户可见角色(用户/系统)忽略
func (w *QA) appendVisible(msg *schema.Message, answered *bool) {
	if msg == nil {
		return
	}
	switch msg.Role {
	case schema.Assistant, schema.Tool:
		w.client.AppendHistory(msg)
		if msg.Role == schema.Assistant && msg.Content != "" {
			*answered = true
		}
	}
}

// logInterrupt 打印一次打断的完整上下文, 用于定位打断来源(哪个 agent/tool 触发)
func (w *QA) logInterrupt(ctx context.Context, event *adk.AgentEvent) {
	info := event.Action.Interrupted
	lg := logx.WithContext(ctx)

	lg.Errorw("qa: agent interrupted",
		logx.Field("session_id", w.client.SessionID),
		logx.Field("agent", event.AgentName),
		logx.Field("run_path", formatRunPath(event.RunPath)),
		logx.Field("context_count", len(info.InterruptContexts)),
	)

	for i, ic := range info.InterruptContexts {
		if ic == nil {
			lg.Errorf("qa: interrupt context[%d] is nil | session=%s", i, w.client.SessionID)
			continue
		}
		parentID := ""
		if ic.Parent != nil {
			parentID = ic.Parent.ID
		}
		rawInfo := ""
		if ic.Info != nil {
			if b, err := json.Marshal(ic.Info); err == nil {
				rawInfo = string(b)
			} else {
				rawInfo = fmt.Sprintf("<marshal error: %v>", err)
			}
		}
		lg.Errorf("qa: interrupt context | session=%s idx=%d id=%s address=%s root_cause=%v parent_id=%s info=%s",
			w.client.SessionID, i, ic.ID, ic.Address.String(), ic.IsRootCause, parentID, rawInfo)
	}
}

// formatRunPath 将 eino 执行路径拼接为可读字符串
func formatRunPath(path []adk.RunStep) string {
	if len(path) == 0 {
		return ""
	}
	parts := make([]string, 0, len(path))
	for i := range path {
		parts = append(parts, path[i].String())
	}
	return strings.Join(parts, "/")
}

// recordInterrupt 记录打断信息, 供 SSE 推送给前端
func (w *QA) recordInterrupt(info *adk.InterruptInfo) {
	if info == nil {
		return
	}
	list := make([]protocol.SSEInterrupt, 0, len(info.InterruptContexts))
	for _, ic := range info.InterruptContexts {
		if ic == nil {
			continue
		}
		item := protocol.SSEInterrupt{
			ID:          ic.ID,
			Address:     ic.Address.String(),
			IsRootCause: ic.IsRootCause,
		}
		if ic.Parent != nil {
			item.ParentID = ic.Parent.ID
		}
		if ic.Info != nil {
			if b, err := json.Marshal(ic.Info); err == nil {
				item.Info = b
			} else {
				logx.Errorf("qa: marshal interrupt info fail | session=%s id=%s err=%v", w.client.SessionID, ic.ID, err)
			}
		}
		list = append(list, item)
	}

	raw, err := json.Marshal(list)
	if err != nil {
		logx.Errorf("qa: marshal interrupt list fail | session=%s err=%v", w.client.SessionID, err)
		return
	}
	msg := json.RawMessage(raw)
	w.client.History.SetInterrupt(&msg)
}

// withoutReasoningAll 复制消息列表并逐条清空思考内容
func withoutReasoningAll(msgs []*schema.Message) []*schema.Message {
	out := make([]*schema.Message, 0, len(msgs))
	for _, m := range msgs {
		out = append(out, withoutReasoning(m))
	}
	return out
}

// withoutReasoning 复制消息并清空思考内容, 不改动原消息
func withoutReasoning(msg *schema.Message) *schema.Message {
	if msg == nil || msg.ReasoningContent == "" {
		return msg
	}
	copied := *msg
	copied.ReasoningContent = ""
	return &copied
}
