package skill

import (
	"context"
	"strings"
	"testing"
)

func TestSupportSkillTool(t *testing.T) {
	tl, err := NewSupportSkillTool()
	if err != nil {
		t.Fatalf("new tool: %v", err)
	}

	info, err := tl.Info(context.Background())
	if err != nil {
		t.Fatalf("info: %v", err)
	}
	if info.Name != SupportSkillName {
		t.Fatalf("name = %q, want %q", info.Name, SupportSkillName)
	}

	out, err := tl.InvokableRun(context.Background(), "{}")
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !strings.Contains(out, "Muti-Document") {
		t.Fatalf("skill doc not loaded, got %q", out)
	}
}
