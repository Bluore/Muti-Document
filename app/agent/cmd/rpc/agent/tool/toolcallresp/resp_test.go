package toolcallresp

import (
	"strings"
	"testing"
)

func TestDocumentContent(t *testing.T) {
	out := DocumentContent("标题", "正文内容", 3)
	for _, want := range []string{"标题", "正文内容", "3"} {
		if !strings.Contains(out, want) {
			t.Fatalf("output %q missing %q", out, want)
		}
	}
}

func TestDocumentListEmpty(t *testing.T) {
	if out := DocumentList(nil); !strings.Contains(out, "还没有创建") {
		t.Fatalf("output %q should mention empty list", out)
	}
}

func TestDocumentConnectionCreatedWithFeedback(t *testing.T) {
	out := DocumentConnectionCreatedWithFeedback("d1", "bot-1", "顺便改一下标题")
	for _, want := range []string{"d1", "bot-1", "顺便改一下标题"} {
		if !strings.Contains(out, want) {
			t.Fatalf("output %q missing %q", out, want)
		}
	}
	if plain := DocumentConnectionCreatedWithFeedback("d1", "bot-1", ""); strings.Contains(plain, "补充说明") {
		t.Fatalf("output %q should not mention feedback when empty", plain)
	}
}

func TestDocumentConnectionRejected(t *testing.T) {
	out := DocumentConnectionRejected("d1", "不想改这篇")
	for _, want := range []string{"d1", "不想改这篇", "不要再次尝试"} {
		if !strings.Contains(out, want) {
			t.Fatalf("output %q missing %q", out, want)
		}
	}

	// 没有补充说明时也应给出明确取消文案
	if out = DocumentConnectionRejected("d1", ""); !strings.Contains(out, "已取消") {
		t.Fatalf("output %q should mention cancellation", out)
	}
}

func TestDocumentListItems(t *testing.T) {
	out := DocumentList([]DocumentItem{
		{ID: "d1", Summary: "第一篇", UpdateAt: 1700000000},
		{ID: "d2"},
	})
	for _, want := range []string{"d1", "第一篇", "d2", "未命名文档", "2"} {
		if !strings.Contains(out, want) {
			t.Fatalf("output %q missing %q", out, want)
		}
	}
}
