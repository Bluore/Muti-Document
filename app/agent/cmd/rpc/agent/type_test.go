package agent

import (
	"encoding/json"
	"strings"
	"testing"
)

func strPtr(s string) *string { return &s }

func TestResumeConditionToNaturalEmpty(t *testing.T) {
	if got := (ResumeCondition{}).ToNatural(); got != nil {
		t.Fatalf("empty condition = %q, want nil", got)
	}
}

func TestResumeConditionToNaturalFull(t *testing.T) {
	ts := int64(1700000000)
	cond := ResumeCondition{
		UserID:       strPtr("u1"),
		UserName:     strPtr("张三"),
		DocumentID:   strPtr("d1"),
		DocumentName: strPtr("设计稿"),
		CursorAnchor: strPtr("120"),
		CursorSelect: strPtr("第一段"),
		Timestamp:    &ts,
		TimeZone:     strPtr("Asia/Shanghai"),
		Language:     strPtr("zh-CN"),
		UserIP:       strPtr("1.2.3.4"),
	}

	want := "【当前上下文】用户「张三」(ID: u1)，正在查看文档「设计稿」(ID: d1)，" +
		"光标锚点位于 120，选中内容「第一段」，当前时间 2023-11-15 06:13:20 (Asia/Shanghai)，" +
		"使用语言 zh-CN，IP 地址 1.2.3.4。"
	if got := string(cond.ToNatural()); got != want {
		t.Fatalf("ToNatural() =\n%q\nwant\n%q", got, want)
	}
}

func TestResumeConditionToNaturalPartial(t *testing.T) {
	cond := ResumeCondition{UserID: strPtr("u1")}
	got := string(cond.ToNatural())
	if !strings.Contains(got, "用户 ID: u1") {
		t.Fatalf("partial condition = %q, want user id", got)
	}
}

func TestResumeConditionTimestampMilliseconds(t *testing.T) {
	sec := int64(1700000000)
	ms := int64(1700000000000)

	want := "当前时间 2023-11-15 06:13:20 (Asia/Shanghai)"
	if got := formatTime(&sec, strPtr("Asia/Shanghai")); got != want {
		t.Fatalf("seconds = %q, want %q", got, want)
	}
	if got := formatTime(&ms, strPtr("Asia/Shanghai")); got != want {
		t.Fatalf("milliseconds = %q, want %q", got, want)
	}
}

func TestResumeConditionJSONFieldNames(t *testing.T) {
	raw := `{"user_id":"u1","user_name":"张三","document_id":"d1","cursor_anchor":"9","timestamp":1700000000,"language":"zh-CN","user_ip":"1.2.3.4"}`

	var cond ResumeCondition
	if err := json.Unmarshal([]byte(raw), &cond); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if cond.UserID == nil || *cond.UserID != "u1" ||
		cond.DocumentID == nil || *cond.DocumentID != "d1" ||
		cond.UserIP == nil || *cond.UserIP != "1.2.3.4" {
		t.Fatalf("unexpected parsed condition: %+v", cond)
	}
}
