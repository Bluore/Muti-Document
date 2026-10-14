package toolcallresp

import (
	"fmt"
	"strings"
	"time"
)

// DocumentItem 文档列表项, 供列表响应组装自然语言
type DocumentItem struct {
	ID       string
	Summary  string
	UpdateAt int64
}

// DocumentContent 文档内容查询成功的自然语言响应
func DocumentContent(summary, content string, version int64) string {
	if summary == "" {
		summary = "未命名文档"
	}
	return fmt.Sprintf("已找到文档《%s》(版本 %d)，正文如下：\n\n%s", summary, version, content)
}

// DocumentList 文档列表查询成功的自然语言响应
func DocumentList(items []DocumentItem) string {
	if len(items) == 0 {
		return "你还没有创建任何文档。"
	}

	var b strings.Builder
	fmt.Fprintf(&b, "你共创建了 %d 篇文档：\n", len(items))
	for i, it := range items {
		summary := it.Summary
		if summary == "" {
			summary = "未命名文档"
		}
		fmt.Fprintf(&b, "%d. 《%s》(ID: %s, 更新时间: %s)\n", i+1, summary, it.ID, formatTime(it.UpdateAt))
	}
	return strings.TrimRight(b.String(), "\n")
}

// formatTime 秒级时间戳格式化为可读时间, 非法值返回未知
func formatTime(sec int64) string {
	if sec <= 0 {
		return "未知"
	}
	return time.Unix(sec, 0).Format("2006-01-02 15:04")
}
