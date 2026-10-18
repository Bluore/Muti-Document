package session

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Bluore/mutidoc/app/agent/cmd/rpc/protocol"
	"github.com/cloudwego/eino/schema"
)

func TestMessagesJSONLRoundTrip(t *testing.T) {
	msgs := []*schema.Message{
		schema.UserMessage("你好"),
		schema.AssistantMessage("hello", nil),
	}

	data, err := EncodeMessages(msgs)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	// JSONL: 每条消息一行, 以换行结尾
	if len(data) == 0 || data[len(data)-1] != '\n' {
		t.Fatalf("want trailing newline, got %q", string(data))
	}

	got, err := DecodeMessages(data)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].Role != schema.User || got[0].Content != "你好" {
		t.Fatalf("unexpected msg0: %+v", got[0])
	}
	if got[1].Role != schema.Assistant || got[1].Content != "hello" {
		t.Fatalf("unexpected msg1: %+v", got[1])
	}
}

func TestEncodeMessagesAppendKeepsPrefixStable(t *testing.T) {
	first := []*schema.Message{schema.UserMessage("a")}
	firstData, err := EncodeMessages(first)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	more := append(append([]*schema.Message{}, first...), schema.AssistantMessage("b", nil))
	moreData, err := EncodeMessages(more)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	if string(moreData[:len(firstData)]) != string(firstData) {
		t.Fatalf("JSONL 追加后前缀应保持不变\nold=%q\nnew=%q", string(firstData), string(moreData))
	}
}

func TestDecodeEmptyMessages(t *testing.T) {
	msgs, err := DecodeMessages([]byte(""))
	if err != nil {
		t.Fatalf("decode empty: %v", err)
	}
	if len(msgs) != 0 {
		t.Fatalf("len = %d, want 0", len(msgs))
	}
}

func TestOutputReadFromOffset(t *testing.T) {
	h := NewOutput()
	user := schema.UserMessage("你好")
	assistant := schema.AssistantMessage("hello", nil)
	h.AppendChunks(user, assistant)

	if len(h.Messages()) != 2 {
		t.Fatalf("messages = %d, want 2", len(h.Messages()))
	}

	data, newOffset := h.ReadFromOffset(0)
	msg := decodeSSE(t, data)

	if msg.Message == nil || len(*msg.Message) != 2 {
		t.Fatalf("msg items = %v, want 2", msg.Message)
	}
	if (*msg.Message)[0].Role != string(schema.User) || (*msg.Message)[0].Content != "你好" {
		t.Fatalf("unexpected item0: %+v", (*msg.Message)[0])
	}
	if (*msg.Message)[1].Role != string(schema.Assistant) || (*msg.Message)[1].Content != "hello" {
		t.Fatalf("unexpected item1: %+v", (*msg.Message)[1])
	}

	want := int64(len("你好") + len("hello"))
	if newOffset != want {
		t.Fatalf("newOffset = %d, want %d", newOffset, want)
	}
}

func TestOutputReadFromOffsetMidMessage(t *testing.T) {
	h := NewOutput()
	h.AppendChunks(schema.UserMessage("你好"), schema.AssistantMessage("hello", nil))

	off := int64(len("你好"))
	data, newOffset := h.ReadFromOffset(off)
	msg := decodeSSE(t, data)

	if msg.Message == nil || len(*msg.Message) != 1 {
		t.Fatalf("msg items = %v, want 1", msg.Message)
	}
	if (*msg.Message)[0].Role != string(schema.Assistant) || (*msg.Message)[0].Content != "hello" {
		t.Fatalf("unexpected item: %+v", (*msg.Message)[0])
	}
	if newOffset != int64(len("你好")+len("hello")) {
		t.Fatalf("newOffset = %d", newOffset)
	}
}

// TestOutputReadFromOffsetNoNew 游标已到末尾时返回空增量
func TestOutputReadFromOffsetNoNew(t *testing.T) {
	h := NewOutput()
	h.AppendChunks(schema.UserMessage("abc"))

	total := int64(len("abc"))
	data, newOffset := h.ReadFromOffset(total)
	msg := decodeSSE(t, data)

	if msg.Message != nil || msg.ToolCall != nil {
		t.Fatalf("want empty payload, got %s", string(data))
	}
	if newOffset != total {
		t.Fatalf("newOffset = %d, want %d", newOffset, total)
	}
}

func TestOutputStreamingMerge(t *testing.T) {
	h := NewOutput()
	h.AppendChunks(schema.AssistantMessage("hel", nil), schema.AssistantMessage("lo", nil))

	if len(h.Messages()) != 1 {
		t.Fatalf("messages = %d, want 1 (分片应合并)", len(h.Messages()))
	}
	if got := h.Messages()[0].Content; got != "hello" {
		t.Fatalf("content = %q, want hello", got)
	}

	data, _ := h.ReadFromOffset(0)
	msg := decodeSSE(t, data)
	if msg.Message == nil || len(*msg.Message) != 1 || (*msg.Message)[0].Content != "hello" {
		t.Fatalf("unexpected payload: %s", string(data))
	}
}

// TestOutputStreamingDeltas 思考文本与正文分片应作为独立增量返回(打字机效果)
func TestOutputStreamingDeltas(t *testing.T) {
	h := NewOutput()
	h.AppendChunks(&schema.Message{Role: schema.Assistant, ReasoningContent: "thin"})

	data, off := h.ReadFromOffset(0)
	item := firstItem(t, data)
	if item.Reasoning != "thin" || item.Content != "" {
		t.Fatalf("first delta = %+v, want reasoning=thin", item)
	}

	// 后续分片: 思考追加一个字符, 正文追加两个字符
	h.AppendChunks(
		&schema.Message{Role: schema.Assistant, ReasoningContent: "k"},
		&schema.Message{Role: schema.Assistant, Content: "hi"},
	)

	data, off2 := h.ReadFromOffset(off)
	msg := decodeSSE(t, data)
	if msg.Message == nil || len(*msg.Message) != 1 {
		t.Fatalf("unexpected payload: %s", string(data))
	}
	got := (*msg.Message)[0]
	if got.Reasoning != "k" || got.Content != "hi" {
		t.Fatalf("delta = %+v, want reasoning=k content=hi", got)
	}

	// 到达末尾后无新增
	if _, end := h.ReadFromOffset(off2); end != off2 {
		t.Fatalf("offset drifted: %d != %d", end, off2)
	}
}

// TestOutputReasoningSeparated 思考文本与正文应分开返回
func TestOutputReasoningSeparated(t *testing.T) {
	h := NewOutput()
	m := schema.AssistantMessage("answer", nil)
	m.ReasoningContent = "think"
	h.AppendChunks(m)

	data, _ := h.ReadFromOffset(0)
	item := firstItem(t, data)
	if item.Reasoning != "think" || item.Content != "answer" {
		t.Fatalf("item = %+v, want reasoning=think content=answer", item)
	}
}

func TestOutputBytesLoadRoundTrip(t *testing.T) {
	h := NewOutput()
	h.AppendChunks(schema.UserMessage("你好"), schema.AssistantMessage("hello", nil))

	h2 := NewOutput()
	if err := h2.Load(h.Bytes()); err != nil {
		t.Fatalf("load: %v", err)
	}

	got := h2.Messages()
	if len(got) != 2 {
		t.Fatalf("messages = %d, want 2", len(got))
	}
	if got[0].Content != "你好" || got[1].Content != "hello" {
		t.Fatalf("unexpected messages: %+v", got)
	}

	data, newOffset := h2.ReadFromOffset(0)
	msg := decodeSSE(t, data)
	if msg.Message == nil || len(*msg.Message) != 2 {
		t.Fatalf("unexpected payload: %s", string(data))
	}
	if newOffset != int64(len("你好")+len("hello")) {
		t.Fatalf("newOffset = %d", newOffset)
	}
}

func TestOutputToolCall(t *testing.T) {
	h := NewOutput()
	assistant := schema.AssistantMessage("", []schema.ToolCall{{
		ID:       "call_1",
		Type:     "function",
		Function: schema.FunctionCall{Name: "search", Arguments: "{}"},
	}})
	h.AppendChunks(assistant)
	h.FinalizeMessage()

	data, newOffset := h.ReadFromOffset(0)
	msg := decodeSSE(t, data)

	if msg.ToolCall == nil || len(*msg.ToolCall) != 1 {
		t.Fatalf("tool_call = %v, want 1", msg.ToolCall)
	}
	if !strings.Contains((*msg.ToolCall)[0].ToolMsg, "search") {
		t.Fatalf("unexpected tool msg: %q", (*msg.ToolCall)[0].ToolMsg)
	}
	if newOffset <= 0 {
		t.Fatalf("newOffset = %d, want > 0", newOffset)
	}

	// 到达末尾后不再重复推送
	if _, off := h.ReadFromOffset(newOffset); off != newOffset {
		t.Fatalf("re-read offset = %d, want %d", off, newOffset)
	}
	data, _ = h.ReadFromOffset(newOffset)
	again := decodeSSE(t, data)
	if again.ToolCall != nil {
		t.Fatalf("tool_call should not be repeated: %s", string(data))
	}
}

func firstItem(t *testing.T, data []byte) protocol.SSEMessageItem {
	t.Helper()
	msg := decodeSSE(t, data)
	if msg.Message == nil || len(*msg.Message) != 1 {
		t.Fatalf("msg items = %v, want 1 (data=%s)", msg.Message, string(data))
	}
	return (*msg.Message)[0]
}

func decodeSSE(t *testing.T, data []byte) protocol.SSEMessage {
	t.Helper()
	var msg protocol.SSEMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		t.Fatalf("unmarshal sse: %v (data=%s)", err, string(data))
	}
	return msg
}
