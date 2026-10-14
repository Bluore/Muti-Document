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
