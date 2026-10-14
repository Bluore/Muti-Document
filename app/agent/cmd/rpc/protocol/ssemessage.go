package protocol

import (
	"encoding/json"

	"github.com/Bluore/mutidoc/pkg/globalkey"
)

// SSEMessageItem 一条对话消息的可见内容增量, 思考文本与正文分开返回
type SSEMessageItem = struct {
	Role      string `json:"role"`
	Content   string `json:"content,omitempty"`
	Reasoning string `json:"reasoning,omitempty"`
}

// SSEToolCall 一次工具调用的增量
type SSEToolCall = struct {
	Offset  int64  `json:"offset"`
	ToolMsg string `json:"tool_msg"`
}

// SSEInterrupt 一次打断的对外描述
type SSEInterrupt struct {
	// ID 打断点唯一地址, 恢复时原样回传
	ID string `json:"id"`
	// Address 组件路径, 如 "agent:DocumentAgent/tool:edit"
	Address string `json:"address"`
	// Info 触发方提供的用户可见信息
	Info json.RawMessage `json:"info,omitempty"`
	// IsRootCause 是否为根因打断点
	IsRootCause bool `json:"is_root_cause"`
	// ParentID 父打断点 ID
	ParentID string `json:"parent_id,omitempty"`
}

// SSEMessage SSE 推送的整包载荷
type SSEMessage struct {
	Offset int64 `json:"-"`
	Length int64 `json:"-"`
	// 按照顺序输出
	Message   *[]SSEMessageItem       `json:"msg"`
	ToolCall  *[]SSEToolCall          `json:"tool_call"`
	Interrupt *json.RawMessage        `json:"interrupt"`
	Status    globalkey.SessionStatus `json:"status"`
}

// payload 序列化影子结构, Offset/Length 仅服务端游标, 不对外输出
type payload struct {
	Message   *[]SSEMessageItem       `json:"msg,omitempty"`
	ToolCall  *[]SSEToolCall          `json:"tool_call,omitempty"`
	Interrupt *json.RawMessage        `json:"interrupt,omitempty"`
	Status    globalkey.SessionStatus `json:"status"`
}

func (s *SSEMessage) MarshalJSON() ([]byte, error) {
	return json.Marshal(payload{
		Message:   s.Message,
		ToolCall:  s.ToolCall,
		Interrupt: s.Interrupt,
		Status:    s.Status,
	})
}

func (s *SSEMessage) UnmarshalJSON(data []byte) error {
	var p payload
	if err := json.Unmarshal(data, &p); err != nil {
		return err
	}
	s.Message = p.Message
	s.ToolCall = p.ToolCall
	s.Interrupt = p.Interrupt
	s.Status = p.Status
	return nil
}
