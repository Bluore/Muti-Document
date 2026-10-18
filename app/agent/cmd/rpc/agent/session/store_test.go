package session

import (
	"testing"

	"github.com/Bluore/mutidoc/pkg/globalkey"
)

func TestGetOrInit(t *testing.T) {
	const id = "test-get-or-init"
	defer Delete(id)

	built := 0
	build := func() (*SessionClient, error) {
		built++
		return NewSessionClient(Session{SessionID: id, UserID: "u1"}), nil
	}

	first, err := GetOrInit(id, build)
	if err != nil {
		t.Fatalf("first GetOrInit: %v", err)
	}
	second, err := GetOrInit(id, build)
	if err != nil {
		t.Fatalf("second GetOrInit: %v", err)
	}
	if first != second {
		t.Fatal("GetOrInit 应返回同一个会话客户端")
	}
	if built != 1 {
		t.Fatalf("build called %d times, want 1", built)
	}
}

func TestTryStartGenerate(t *testing.T) {
	client := NewSessionClient(Session{SessionID: "test-generate"})

	if !client.tryStartGenerate() {
		t.Fatal("首次应能进入生成状态")
	}
	if client.Status() != globalkey.SessionStatusGenerating {
		t.Fatalf("status = %d", client.Status())
	}
	if client.tryStartGenerate() {
		t.Fatal("生成中不应再次进入")
	}

	client.setStatus(globalkey.SessionStatusIdle)
	if !client.tryStartGenerate() {
		t.Fatal("空闲后应能再次进入生成状态")
	}
}
