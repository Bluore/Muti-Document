package workflow

import (
	"context"
	"errors"
	"testing"

	"github.com/Bluore/mutidoc/app/agent/cmd/rpc/agent/session"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

// stubAgent 固定回复一个字符串, 用于测试问答工作流
type stubAgent struct {
	reply string
}

func (s *stubAgent) Name(context.Context) string        { return "stub" }
func (s *stubAgent) Description(context.Context) string { return "stub agent" }

func (s *stubAgent) Run(_ context.Context, _ *adk.AgentInput, _ ...adk.AgentRunOption) *adk.AsyncIterator[*adk.AgentEvent] {
	iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()
	gen.Send(adk.EventFromMessage(schema.AssistantMessage(s.reply, nil), nil, schema.Assistant, ""))
	gen.Close()
	return iter
}

func newTestQA(t *testing.T, reply string) (*QA, *session.SessionClient) {
	t.Helper()

	ctx := context.Background()
	ag := &stubAgent{reply: reply}
	runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: ag})

	client := session.NewSessionClient(session.Session{SessionID: "s1", UserID: "u1"})
	client.SetRuntime(runner, ag, nil, "")

	return NewQA(client), client
}

func TestQAAsk(t *testing.T) {
	qa, client := newTestQA(t, "hello")

	if err := qa.Ask(context.Background(), "hi"); err != nil {
		t.Fatalf("ask: %v", err)
	}

	history := client.SnapshotHistory()
	if len(history) != 2 {
		t.Fatalf("history len = %d, want 2", len(history))
	}
	if history[0].Role != schema.User || history[0].Content != "hi" {
		t.Fatalf("unexpected user message: %+v", history[0])
	}
	if history[1].Role != schema.Assistant || history[1].Content != "hello" {
		t.Fatalf("unexpected assistant message: %+v", history[1])
	}

	memory := client.SnapshotMemory()
	if len(memory) != 2 {
		t.Fatalf("memory len = %d, want 2", len(memory))
	}
}

func TestWithoutReasoning(t *testing.T) {
	msg := schema.AssistantMessage("answer", nil)
	msg.ReasoningContent = "thinking"

	stripped := withoutReasoning(msg)
	if stripped.ReasoningContent != "" {
		t.Fatalf("reasoning content not stripped: %q", stripped.ReasoningContent)
	}
	if stripped.Content != "answer" {
		t.Fatalf("content changed: %q", stripped.Content)
	}
	// 原消息不受影响
	if msg.ReasoningContent != "thinking" {
		t.Fatalf("original message mutated: %q", msg.ReasoningContent)
	}
}

func TestQAAskNilSession(t *testing.T) {
	qa := NewQA(nil)
	if err := qa.Ask(context.Background(), "hi"); !errors.Is(err, ErrNilSession) {
		t.Fatalf("err = %v, want %v", err, ErrNilSession)
	}
}

func TestQAAskClosedSession(t *testing.T) {
	qa, client := newTestQA(t, "hello")
	client.Close()

	if err := qa.Ask(context.Background(), "hi"); !errors.Is(err, session.ErrSessionClosed) {
		t.Fatalf("err = %v, want %v", err, session.ErrSessionClosed)
	}
}

// streamAgent 以流式事件吐出若干分片
type streamAgent struct {
	chunks []*schema.Message
}

func (s *streamAgent) Name(context.Context) string        { return "stream" }
func (s *streamAgent) Description(context.Context) string { return "stream agent" }

func (s *streamAgent) Run(_ context.Context, _ *adk.AgentInput, _ ...adk.AgentRunOption) *adk.AsyncIterator[*adk.AgentEvent] {
	iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()
	gen.Send(adk.EventFromMessage(nil, schema.StreamReaderFromArray(s.chunks), schema.Assistant, ""))
	gen.Close()
	return iter
}

func TestQAAskStreaming(t *testing.T) {
	ctx := context.Background()
	ag := &streamAgent{chunks: []*schema.Message{
		{Role: schema.Assistant, ReasoningContent: "think"},
		{Role: schema.Assistant, Content: "hel"},
		{Role: schema.Assistant, Content: "lo"},
	}}
	runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: ag})

	client := session.NewSessionClient(session.Session{SessionID: "s2", UserID: "u1"})
	client.SetRuntime(runner, ag, nil, "")

	if err := NewQA(client).Ask(ctx, "hi"); err != nil {
		t.Fatalf("ask: %v", err)
	}

	history := client.SnapshotHistory()
	if len(history) != 2 {
		t.Fatalf("history len = %d, want 2", len(history))
	}
	assistant := history[1]
	if assistant.Role != schema.Assistant || assistant.Content != "hello" {
		t.Fatalf("assistant = %+v", assistant)
	}
	if assistant.ReasoningContent != "think" {
		t.Fatalf("reasoning = %q, want think", assistant.ReasoningContent)
	}

	// 模型记忆去掉思考内容
	memory := client.SnapshotMemory()
	if len(memory) != 2 {
		t.Fatalf("memory len = %d, want 2", len(memory))
	}
	if memory[1].ReasoningContent != "" || memory[1].Content != "hello" {
		t.Fatalf("memory assistant = %+v", memory[1])
	}
}
