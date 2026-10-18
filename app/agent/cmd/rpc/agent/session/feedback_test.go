package session

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/Bluore/mutidoc/app/agent/cmd/rpc/agent/approval"
	"github.com/Bluore/mutidoc/app/agent/cmd/rpc/protocol"
	"github.com/Bluore/mutidoc/pkg/globalkey"
)

func TestResumeTargets(t *testing.T) {
	client := NewSessionClient(Session{SessionID: "s-feedback"})
	decision := approval.Decision{Approved: true, Feedback: "同意"}

	client.setPendingInterrupts([]protocol.SSEInterrupt{
		{ID: "root-1", IsRootCause: true},
		{ID: "child-1"},
	})
	targets := client.resumeTargets(decision)
	if len(targets) != 1 {
		t.Fatalf("targets = %d, want 1", len(targets))
	}
	got, ok := targets["root-1"]
	if !ok {
		t.Fatalf("root interrupt should be targeted, got %v", targets)
	}
	d, ok := got.(*approval.Decision)
	if !ok || d == nil || !d.Approved || d.Feedback != "同意" {
		t.Fatalf("unexpected decision: %#v", got)
	}

	// 没有根因标记时退化为恢复全部打断点
	client.setPendingInterrupts([]protocol.SSEInterrupt{{ID: "a"}, {ID: "b"}})
	if targets = client.resumeTargets(decision); len(targets) != 2 {
		t.Fatalf("fallback targets = %d, want 2", len(targets))
	}
}

func TestHumanFeedbackNotInterrupted(t *testing.T) {
	client := NewSessionClient(Session{SessionID: "s-feedback"})
	if err := client.HumanFeedback(true, ""); !errors.Is(err, ErrNotInterrupted) {
		t.Fatalf("err = %v, want ErrNotInterrupted", err)
	}

	// 状态为已打断但没有待处理打断点, 同样不可恢复
	client.setStatus(globalkey.SessionStatusInterrupted)
	if err := client.HumanFeedback(true, ""); !errors.Is(err, ErrNotInterrupted) {
		t.Fatalf("err = %v, want ErrNotInterrupted", err)
	}
}

func TestHumanFeedbackGenerating(t *testing.T) {
	client := NewSessionClient(Session{SessionID: "s-feedback"})
	client.setStatus(globalkey.SessionStatusGenerating)
	if err := client.HumanFeedback(true, ""); !errors.Is(err, ErrGenerating) {
		t.Fatalf("err = %v, want ErrGenerating", err)
	}
}

func TestClearInterrupt(t *testing.T) {
	client := NewSessionClient(Session{SessionID: "s-feedback"})
	client.setPendingInterrupts([]protocol.SSEInterrupt{{ID: "root-1", IsRootCause: true}})
	raw := json.RawMessage(`[{"id":"root-1"}]`)
	client.Output.SetInterrupt(&raw)

	client.clearInterrupt()

	if client.hasPendingInterrupt() {
		t.Fatal("pending interrupt should be cleared")
	}
	data, _ := client.OutputSSE(0)
	if len(data) == 0 || containsInterrupt(data) {
		t.Fatalf("interrupt should be omitted from SSE payload: %s", string(data))
	}
}

// containsInterrupt 判断 SSE 载荷中是否带有 interrupt 字段
func containsInterrupt(data []byte) bool {
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(data, &payload); err != nil {
		return true
	}
	_, ok := payload["interrupt"]
	return ok
}

// TestGenerateRejectedWhileInterrupted 有未处理打断时不允许直接开新一轮
func TestGenerateRejectedWhileInterrupted(t *testing.T) {
	client := NewSessionClient(Session{SessionID: "s-feedback"})
	client.setStatus(globalkey.SessionStatusInterrupted)
	if err := client.Generate("hi"); !errors.Is(err, ErrInterrupted) {
		t.Fatalf("err = %v, want ErrInterrupted", err)
	}
}

// TestNewSessionClientCarriesApprovalTracker 会话上下文应挂载审批白名单
func TestNewSessionClientCarriesApprovalTracker(t *testing.T) {
	client := NewSessionClient(Session{SessionID: "s-feedback", UserID: "u1"})
	ctx := client.context()
	approval.Allow(ctx, "d1")
	if !approval.Allowed(ctx, "d1") {
		t.Fatal("approval tracker should be attached to session ctx")
	}
}
