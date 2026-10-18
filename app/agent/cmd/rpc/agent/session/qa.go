package session

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/Bluore/mutidoc/app/agent/cmd/rpc/protocol"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
	"github.com/zeromicro/go-zero/core/logx"
)

// 一次问答工作流错误
var (
	ErrNilSession = errors.New("session client is nil")
	ErrNoAnswer   = errors.New("agent produced no answer")
)

// generateTurn 执行一轮问答: 逐 chunk 写入可见输出(打字机效果), 并把完整消息追加到模型上下文
func (c *SessionClient) generateTurn(ctx context.Context, query string) error {
	if c == nil {
		return ErrNilSession
	}
	if c.isClosed() {
		return ErrSessionClosed
	}

	logx.WithContext(ctx).Infow("qa: turn start",
		logx.Field("session_id", c.SessionID),
		logx.Field("query_len", len(query)),
	)

	// 用户消息写入可见输出, 同时追加到模型上下文
	c.appendOutput(schema.UserMessage(query))
	c.appendContext(schema.UserMessage(query))

	iter, err := c.run(ctx)
	if err != nil {
		logx.WithContext(ctx).Errorf("qa: run fail | session=%s err=%v", c.SessionID, err)
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
				c.SessionID, event.AgentName, formatRunPath(event.RunPath), event.Err)
			return event.Err
		}
		if event.Action != nil && event.Action.Interrupted != nil {
			c.logInterrupt(ctx, event)
			c.recordInterrupt(event.Action.Interrupted)
			return ErrInterrupted
		}
		if event.Output == nil || event.Output.MessageOutput == nil {
			continue
		}

		mo := event.Output.MessageOutput
		if mo.IsStreaming && mo.MessageStream != nil {
			// 流式: 每个分片即时写入可见输出, SSE 轮询即可增量返回
			if err = c.stream(mo.MessageStream, mo.Role, &answered); err != nil {
				logx.WithContext(ctx).Errorf("qa: consume stream fail | session=%s agent=%s err=%v",
					c.SessionID, event.AgentName, err)
				return err
			}
			c.Output.FinalizeMessage()
			continue
		}
		if mo.Message != nil {
			if c.appendVisible(mo.Message, &answered) {
				c.appendContext(withoutReasoning(mo.Message))
			}
			c.Output.FinalizeMessage()
		}
	}

	if !answered {
		logx.WithContext(ctx).Errorf("qa: no answer produced | session=%s", c.SessionID)
		return ErrNoAnswer
	}

	// 模型上下文只追加, 本轮消息已在生成过程中逐条追加
	logx.WithContext(ctx).Infow("qa: turn done",
		logx.Field("session_id", c.SessionID),
		logx.Field("output_len", len(c.snapshotOutput())),
		logx.Field("context_len", len(c.snapshotContext())),
	)
	return nil
}

// run 以当前上下文作为输入驱动一轮生成
func (c *SessionClient) run(ctx context.Context) (*adk.AsyncIterator[*adk.AgentEvent], error) {
	if c == nil {
		return nil, ErrNilSession
	}
	runner, checkpointID, err := c.runtime()
	if err != nil {
		return nil, err
	}
	return runner.Run(ctx, c.snapshotContext(), adk.WithCheckPointID(checkpointID)), nil
}

// stream 逐分片消费一条流式消息, 结束后把完整消息追加到模型上下文
func (c *SessionClient) stream(stream *schema.StreamReader[*schema.Message], role schema.RoleType, answered *bool) error {
	var full *schema.Message
	for {
		chunk, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
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
		if c.appendVisible(chunk, answered) {
			full = concatMessage(full, chunk)
		}
	}
	if full != nil {
		c.appendContext(withoutReasoning(full))
	}
	return nil
}

// appendVisible 追加一条可见消息, 非用户可见角色(用户/系统)忽略, 返回是否追加
func (c *SessionClient) appendVisible(msg *schema.Message, answered *bool) bool {
	if msg == nil {
		return false
	}
	switch msg.Role {
	case schema.Assistant, schema.Tool:
		c.appendOutput(msg)
		if msg.Role == schema.Assistant && msg.Content != "" {
			*answered = true
		}
		return true
	}
	return false
}

// concatMessage 合并流式分片; 合并失败时保留已有内容
func concatMessage(full, chunk *schema.Message) *schema.Message {
	if full == nil {
		return chunk
	}
	merged, err := schema.ConcatMessages([]*schema.Message{full, chunk})
	if err != nil {
		logx.Errorf("qa: concat stream chunk fail: %v", err)
		return full
	}
	return merged
}

// logInterrupt 打印一次打断的完整上下文, 用于定位打断来源(哪个 agent/tool 触发)
func (c *SessionClient) logInterrupt(ctx context.Context, event *adk.AgentEvent) {
	info := event.Action.Interrupted
	lg := logx.WithContext(ctx)

	lg.Errorw("qa: agent interrupted",
		logx.Field("session_id", c.SessionID),
		logx.Field("agent", event.AgentName),
		logx.Field("run_path", formatRunPath(event.RunPath)),
		logx.Field("context_count", len(info.InterruptContexts)),
	)

	for i, ic := range info.InterruptContexts {
		if ic == nil {
			lg.Errorf("qa: interrupt context[%d] is nil | session=%s", i, c.SessionID)
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
			c.SessionID, i, ic.ID, ic.Address.String(), ic.IsRootCause, parentID, rawInfo)
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
func (c *SessionClient) recordInterrupt(info *adk.InterruptInfo) {
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
				logx.Errorf("qa: marshal interrupt info fail | session=%s id=%s err=%v", c.SessionID, ic.ID, err)
			}
		}
		list = append(list, item)
	}

	raw, err := json.Marshal(list)
	if err != nil {
		logx.Errorf("qa: marshal interrupt list fail | session=%s err=%v", c.SessionID, err)
		return
	}
	msg := json.RawMessage(raw)
	c.Output.SetInterrupt(&msg)
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
