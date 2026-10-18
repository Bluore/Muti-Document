package session

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"sort"
	"sync"

	"github.com/Bluore/mutidoc/app/agent/cmd/rpc/protocol"
	"github.com/Bluore/mutidoc/pkg/globalkey"
	"github.com/cloudwego/eino/schema"
	"github.com/zeromicro/go-zero/core/logx"
)

// EncodeMessage 将单条消息序列化为一行 JSON (以换行结尾)
func EncodeMessage(m *schema.Message) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(m); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// EncodeMessages 将消息序列化为 JSONL, 每条消息一行
func EncodeMessages(msgs []*schema.Message) ([]byte, error) {
	var buf bytes.Buffer
	for _, m := range msgs {
		if m == nil {
			continue
		}
		b, err := EncodeMessage(m)
		if err != nil {
			return nil, err
		}
		buf.Write(b)
	}
	return buf.Bytes(), nil
}

// DecodeMessages 解析 JSONL 消息
func DecodeMessages(data []byte) ([]*schema.Message, error) {
	msgs := make([]*schema.Message, 0)
	dec := json.NewDecoder(bytes.NewReader(data))
	for {
		var m schema.Message
		if err := dec.Decode(&m); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
		msgs = append(msgs, &m)
	}
	return msgs, nil
}

// 时间线条目类型
type entryKind uint8

const (
	entryText     entryKind = iota // 可见文本消息
	entryToolCall                  // 工具调用
)

// entry 时间线上的一项, 在 content 中占据 [start, end) 字节
type entry struct {
	kind  entryKind
	start int64
	end   int64
	// 仅 entryText: 思考文本字节数与消息角色
	reasonLen int64
	role      schema.RoleType
}

// Output 会话输出(用户界面可见), 以文本消息与工具调用组成的时间线并维护字节游标,
// 支持按字节 offset 二分定位并输出增量 SSE 载荷; 只追加, 不覆盖
type Output struct {
	mu       sync.Mutex
	messages []*schema.Message
	entries  []entry
	// content 时间线各项内容的字节拼接(文本消息为可见文本, 工具调用为其 JSON)
	content   []byte
	toolIDs   map[string]struct{}
	interrupt *json.RawMessage
	status    globalkey.SessionStatus
}

// NewOutput 创建空输出
func NewOutput() *Output {
	return &Output{toolIDs: make(map[string]struct{})}
}

// AppendChunks 追加消息(流式分片会与上一条同角色文本消息合并)
func (h *Output) AppendChunks(chunks ...*schema.Message) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for _, msg := range chunks {
		h.appendChunk(msg)
	}
}

func (h *Output) appendChunk(chunk *schema.Message) {
	if chunk == nil {
		logx.Error("output: append nil message")
		return
	}

	if n := len(h.messages); n > 0 {
		if e := h.lastTextEntry(); e != nil {
			if merged, err := schema.ConcatMessages([]*schema.Message{h.messages[n-1], chunk}); err == nil {
				// 合并成功视为同一条消息的流式分片, 重算其可见文本
				h.messages[n-1] = merged
				h.content = append(h.content[:int(e.start)], merged.ReasoningContent...)
				h.content = append(h.content, merged.Content...)
				e.end = int64(len(h.content))
				e.reasonLen = int64(len(merged.ReasoningContent))
				return
			}
		}
	}

	h.appendText(chunk)
}

// lastTextEntry 若最后一项是文本消息则返回其指针
func (h *Output) lastTextEntry() *entry {
	n := len(h.entries)
	if n == 0 || h.entries[n-1].kind != entryText {
		return nil
	}
	return &h.entries[n-1]
}

// appendText 追加一条可见文本消息, 思考文本在前正文在后
func (h *Output) appendText(m *schema.Message) {
	h.messages = append(h.messages, m)

	start := int64(len(h.content))
	h.content = append(h.content, m.ReasoningContent...)
	h.content = append(h.content, m.Content...)
	h.entries = append(h.entries, entry{
		kind:      entryText,
		start:     start,
		end:       int64(len(h.content)),
		reasonLen: int64(len(m.ReasoningContent)),
		role:      m.Role,
	})
}

// FinalizeMessage 标记最后一条消息已完整, 记录其携带的工具调用
func (h *Output) FinalizeMessage() {
	h.mu.Lock()
	defer h.mu.Unlock()

	n := len(h.entries)
	if n == 0 || h.entries[n-1].kind != entryText || len(h.messages) == 0 {
		return
	}
	h.recordToolCalls(h.messages[len(h.messages)-1])
}

// recordToolCalls 追加消息携带的工具调用, 按 ID 去重
func (h *Output) recordToolCalls(m *schema.Message) {
	if m == nil || len(m.ToolCalls) == 0 {
		return
	}
	for _, tc := range m.ToolCalls {
		if tc.ID != "" {
			if _, ok := h.toolIDs[tc.ID]; ok {
				continue
			}
			h.toolIDs[tc.ID] = struct{}{}
		}
		start := int64(len(h.content))
		h.content = append(h.content, toolCallBytes(tc)...)
		h.entries = append(h.entries, entry{
			kind:  entryToolCall,
			start: start,
			end:   int64(len(h.content)),
		})
	}
}

// Messages 返回可见输出消息副本
func (h *Output) Messages() []*schema.Message {
	h.mu.Lock()
	defer h.mu.Unlock()

	out := make([]*schema.Message, len(h.messages))
	copy(out, h.messages)
	return out
}

// Bytes 序列化为 JSONL, 用于落库存储
func (h *Output) Bytes() []byte {
	h.mu.Lock()
	defer h.mu.Unlock()

	data, err := EncodeMessages(h.messages)
	if err != nil {
		logx.Errorf("output: encode fail: %v", err)
		return nil
	}
	return data
}

// Load 从 JSONL 追加输出消息, 重建时间线与字节游标; 只追加, 不覆盖已有内容
func (h *Output) Load(data []byte) error {
	msgs, err := DecodeMessages(data)
	if err != nil {
		return err
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	if h.toolIDs == nil {
		h.toolIDs = make(map[string]struct{})
	}
	for _, m := range msgs {
		if m == nil {
			continue
		}
		h.appendText(m)
		h.recordToolCalls(m)
	}
	return nil
}

// SetInterrupt 记录最近一次打断信息(已序列化的 []SSEInterrupt)
func (h *Output) SetInterrupt(raw *json.RawMessage) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.interrupt = raw
}

// setStatus 同步会话状态, 供 SSE 载荷输出
func (h *Output) setStatus(status globalkey.SessionStatus) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.status = status
}

// ReadFromOffset 从字节 offset 处读取增量, 返回 SSE 载荷与新游标
func (h *Output) ReadFromOffset(offset int64) ([]byte, int64) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if offset < 0 {
		offset = 0
	}
	total := int64(len(h.content))
	if offset > total {
		offset = total
	}

	// 二分定位第一个 end > offset 的条目
	idx := sort.Search(len(h.entries), func(i int) bool { return h.entries[i].end > offset })

	msgs := make([]protocol.SSEMessageItem, 0, len(h.entries))
	calls := make([]protocol.SSEToolCall, 0)
	for ; idx < len(h.entries); idx++ {
		e := h.entries[idx]
		from := offset
		if from < e.start {
			from = e.start
		}
		if from >= e.end {
			continue
		}

		switch e.kind {
		case entryText:
			reasonEnd := e.start + e.reasonLen
			item := protocol.SSEMessageItem{Role: string(e.role)}
			if rs := max64(from, e.start); rs < reasonEnd {
				item.Reasoning = string(h.content[rs:reasonEnd])
			}
			if cs := max64(from, reasonEnd); cs < e.end {
				item.Content = string(h.content[cs:e.end])
			}
			if item.Reasoning != "" || item.Content != "" {
				msgs = append(msgs, item)
			}
		case entryToolCall:
			calls = append(calls, protocol.SSEToolCall{
				Offset:  e.start,
				ToolMsg: string(h.content[e.start:e.end]),
			})
		}
	}

	return h.marshalSSE(msgs, calls, offset, total), total
}

// marshalSSE 组装并序列化 SSE 载荷
func (h *Output) marshalSSE(msgs []protocol.SSEMessageItem, calls []protocol.SSEToolCall, offset, total int64) []byte {
	msg := protocol.SSEMessage{
		Offset:    offset,
		Length:    total,
		Interrupt: h.interrupt,
		Status:    h.status,
	}
	if len(msgs) > 0 {
		msg.Message = &msgs
	}
	if len(calls) > 0 {
		msg.ToolCall = &calls
	}

	data, err := json.Marshal(&msg)
	if err != nil {
		logx.Errorf("history: marshal sse fail: %v", err)
		return nil
	}
	return data
}

// max64 取较大值
func max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

// toolCallBytes 工具调用序列化, 失败时退化为函数名
func toolCallBytes(tc schema.ToolCall) []byte {
	b, err := json.Marshal(tc)
	if err != nil {
		return []byte(tc.Function.Name)
	}
	return b
}
