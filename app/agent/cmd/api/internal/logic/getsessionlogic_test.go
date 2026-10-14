package logic

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Bluore/mutidoc/pkg/globalkey"
)

func TestWriteSSE(t *testing.T) {
	var b strings.Builder
	if err := writeSSE(&b, `{"msg":1}`, 42); err != nil {
		t.Fatalf("writeSSE: %v", err)
	}

	want := "id: 42\ndata: {\"msg\":1}\n\n"
	if b.String() != want {
		t.Fatalf("payload = %q, want %q", b.String(), want)
	}
}

func TestWriteSSEEmpty(t *testing.T) {
	var b strings.Builder
	if err := writeSSE(&b, "", 42); err != nil {
		t.Fatalf("writeSSE: %v", err)
	}
	if b.Len() != 0 {
		t.Fatalf("empty payload should write nothing, got %q", b.String())
	}
}

func TestWriteSSEMultiline(t *testing.T) {
	var b strings.Builder
	if err := writeSSE(&b, "a\nb\n", 1); err != nil {
		t.Fatalf("writeSSE: %v", err)
	}

	want := "id: 1\ndata: a\ndata: b\n\n"
	if b.String() != want {
		t.Fatalf("payload = %q, want %q", b.String(), want)
	}
}

func TestWriteHeartbeat(t *testing.T) {
	var b strings.Builder
	if err := writeHeartbeat(&b); err != nil {
		t.Fatalf("writeHeartbeat: %v", err)
	}
	if b.String() != ": heartbeat\n\n" {
		t.Fatalf("heartbeat = %q", b.String())
	}
}

func TestInitialOffset(t *testing.T) {
	tests := []struct {
		name   string
		url    string
		lastID string
		want   int64
	}{
		{name: "none", url: "/session/stream/1", want: 0},
		{name: "query", url: "/session/stream/1?offset=120", want: 120},
		{name: "last-event-id", url: "/session/stream/1", lastID: "88", want: 88},
		{name: "query wins", url: "/session/stream/1?offset=7", lastID: "88", want: 7},
		{name: "negative ignored", url: "/session/stream/1?offset=-3", want: 0},
		{name: "invalid ignored", url: "/session/stream/1?offset=abc", lastID: "5", want: 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.url, nil)
			if tt.lastID != "" {
				req.Header.Set("Last-Event-ID", tt.lastID)
			}
			if got := initialOffset(req); got != tt.want {
				t.Fatalf("initialOffset = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestSessionPollDuration(t *testing.T) {
	if got := sessionPollDuration(globalkey.SessionStatusGenerating); got != sessionPollInterval {
		t.Fatalf("generating = %v, want %v", got, sessionPollInterval)
	}
	if got := sessionPollDuration(globalkey.SessionStatusIdle); got != sessionIdlePollInterval {
		t.Fatalf("idle = %v, want %v", got, sessionIdlePollInterval)
	}
	if got := sessionPollDuration(globalkey.SessionStatusDone); got != sessionIdlePollInterval {
		t.Fatalf("done = %v, want %v", got, sessionIdlePollInterval)
	}
}
