package session

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Bluore/mutidoc/pkg/globalkey"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

// stubAgent 固定回复一个字符串, 用于测试一轮问答
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

func newTestClient(t *testing.T, reply string) *SessionClient {
	t.Helper()

	ctx := context.Background()
	ag := &stubAgent{reply: reply}
	runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: ag})

	client := NewSessionClient(Session{SessionID: "s1", UserID: "u1"})
	client.setRuntime(runner, ag, nil, "")
	return client
}

// waitGenerate 等待后台生成结束
func waitGenerate(t *testing.T, client *SessionClient) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if client.Status() != globalkey.SessionStatusGenerating {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("generate timeout, status=%d", client.Status())
}

func TestGenerate(t *testing.T) {
	client := newTestClient(t, "hello")

	if err := client.Generate("hi"); err != nil {
		t.Fatalf("generate: %v", err)
	}
	waitGenerate(t, client)

	output := client.snapshotOutput()
	if len(output) != 2 {
		t.Fatalf("output len = %d, want 2", len(output))
	}
	if output[0].Role != schema.User || output[0].Content != "hi" {
		t.Fatalf("unexpected user message: %+v", output[0])
	}
	if output[1].Role != schema.Assistant || output[1].Content != "hello" {
		t.Fatalf("unexpected assistant message: %+v", output[1])
	}

	ctxMsgs := client.snapshotContext()
	if len(ctxMsgs) != 2 {
		t.Fatalf("context len = %d, want 2", len(ctxMsgs))
	}
}

// TestGenerateAppendsContext 多轮对话上下文只追加, 不覆盖上一轮
func TestGenerateAppendsContext(t *testing.T) {
	client := newTestClient(t, "hello")

	for i := 0; i < 2; i++ {
		if err := client.Generate("hi"); err != nil {
			t.Fatalf("generate round %d: %v", i, err)
		}
		waitGenerate(t, client)
	}

	output := client.snapshotOutput()
	if len(output) != 4 {
		t.Fatalf("output len = %d, want 4", len(output))
	}
	ctxMsgs := client.snapshotContext()
	if len(ctxMsgs) != 4 {
		t.Fatalf("context len = %d, want 4", len(ctxMsgs))
	}
	for i, want := range []schema.RoleType{schema.User, schema.Assistant, schema.User, schema.Assistant} {
		if ctxMsgs[i].Role != want {
			t.Fatalf("context[%d].Role = %s, want %s", i, ctxMsgs[i].Role, want)
		}
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

func TestGenerateNilSession(t *testing.T) {
	var client *SessionClient
	if err := client.Generate("hi"); !errors.Is(err, ErrNilSession) {
		t.Fatalf("err = %v, want %v", err, ErrNilSession)
	}
}

func TestGenerateClosedSession(t *testing.T) {
	client := newTestClient(t, "hello")
	client.Close()

	if err := client.Generate("hi"); !errors.Is(err, ErrSessionClosed) {
		t.Fatalf("err = %v, want %v", err, ErrSessionClosed)
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

func TestGenerateStreaming(t *testing.T) {
	ctx := context.Background()
	ag := &streamAgent{chunks: []*schema.Message{
		{Role: schema.Assistant, ReasoningContent: "think"},
		{Role: schema.Assistant, Content: "hel"},
		{Role: schema.Assistant, Content: "lo"},
	}}
	runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: ag})

	client := NewSessionClient(Session{SessionID: "s2", UserID: "u1"})
	client.setRuntime(runner, ag, nil, "")

	if err := client.Generate("hi"); err != nil {
		t.Fatalf("generate: %v", err)
	}
	waitGenerate(t, client)

	output := client.snapshotOutput()
	if len(output) != 2 {
		t.Fatalf("output len = %d, want 2", len(output))
	}
	assistant := output[1]
	if assistant.Role != schema.Assistant || assistant.Content != "hello" {
		t.Fatalf("assistant = %+v", assistant)
	}
	if assistant.ReasoningContent != "think" {
		t.Fatalf("reasoning = %q, want think", assistant.ReasoningContent)
	}

	// 模型上下文去掉思考内容
	ctxMsgs := client.snapshotContext()
	if len(ctxMsgs) != 2 {
		t.Fatalf("context len = %d, want 2", len(ctxMsgs))
	}
	if ctxMsgs[1].ReasoningContent != "" || ctxMsgs[1].Content != "hello" {
		t.Fatalf("context assistant = %+v", ctxMsgs[1])
	}
}
