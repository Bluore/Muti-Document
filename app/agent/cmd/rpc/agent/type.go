package agent

import (
	"fmt"
	"strings"
	"time"
)

type ResumeCondition struct {
	UserID       *string `json:"user_id,omitempty"`
	UserName     *string `json:"user_name,omitempty"`
	DocumentID   *string `json:"document_id,omitempty"`
	DocumentName *string `json:"document_name,omitempty"`
	CursorAnchor *string `json:"cursor_anchor,omitempty"`
	CursorSelect *string `json:"cursor_select,omitempty"`
	Timestamp    *int64  `json:"timestamp,omitempty"`
	TimeZone     *string `json:"time_zone,omitempty"`
	Language     *string `json:"language,omitempty"`
	UserIP       *string `json:"user_ip,omitempty"`
}

// ToNatural 将条件整理为一段自然语言上下文, 供拼接到用户提问前送入模型
func (r ResumeCondition) ToNatural() []byte {
	parts := make([]string, 0, 7)

	if s := formatEntity(r.UserName, r.UserID, "用户"); s != "" {
		parts = append(parts, s)
	}
	if s := formatEntity(r.DocumentName, r.DocumentID, "文档"); s != "" {
		parts = append(parts, "正在查看"+s)
	}
	if v := trimPtr(r.CursorAnchor); v != "" {
		parts = append(parts, "光标锚点位于 "+v)
	}
	if v := trimPtr(r.CursorSelect); v != "" {
		parts = append(parts, "选中内容「"+v+"」")
	}
	if s := formatTime(r.Timestamp, r.TimeZone); s != "" {
		parts = append(parts, s)
	}
	if v := trimPtr(r.Language); v != "" {
		parts = append(parts, "使用语言 "+v)
	}
	if v := trimPtr(r.UserIP); v != "" {
		parts = append(parts, "IP 地址 "+v)
	}

	if len(parts) == 0 {
		return nil
	}
	return []byte("<context>\n[用户上下文]" + strings.Join(parts, "，") + "\n</context>")
}

// trimPtr 安全读取字符串指针并去除首尾空白
func trimPtr(p *string) string {
	if p == nil {
		return ""
	}
	return strings.TrimSpace(*p)
}

// formatEntity 拼接实体名称与 ID, kind 为实体类型(如 "用户"/"文档")
func formatEntity(name, id *string, kind string) string {
	n, i := trimPtr(name), trimPtr(id)
	switch {
	case n != "" && i != "":
		return fmt.Sprintf("%s「%s」(ID: %s)", kind, n, i)
	case n != "":
		return fmt.Sprintf("%s「%s」", kind, n)
	case i != "":
		return fmt.Sprintf("%s ID: %s", kind, i)
	default:
		return ""
	}
}

// formatTime 将时间戳(秒或毫秒)与时区格式化为可读时间
func formatTime(ts *int64, zone *string) string {
	z := trimPtr(zone)
	if ts == nil {
		if z == "" {
			return ""
		}
		return "时区 " + z
	}

	t := time.Unix(*ts, 0)
	if *ts > 1e12 { // 毫秒时间戳
		t = time.UnixMilli(*ts)
	}
	if z != "" {
		if loc, err := time.LoadLocation(z); err == nil {
			t = t.In(loc)
		}
		return fmt.Sprintf("当前时间 %s (%s)", t.Format("2006-01-02 15:04:05"), z)
	}
	return "当前时间 " + t.Format("2006-01-02 15:04:05")
}
