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

// DocumentChunkItem 向量检索命中的一条文档分块, 供检索响应组装自然语言
type DocumentChunkItem struct {
	DocID      string
	ChunkIndex int64
	Text       string
	Score      float32
}

// DocumentSearchHits 向量检索成功的自然语言响应
func DocumentSearchHits(items []DocumentChunkItem) string {
	if len(items) == 0 {
		return "没有检索到相关内容。"
	}

	var b strings.Builder
	fmt.Fprintf(&b, "检索到 %d 条相关内容：\n", len(items))
	for i, it := range items {
		fmt.Fprintf(&b, "\n%d. (文档 ID: %s, 分块序号: %d, 相似度: %.4f)\n%s\n",
			i+1, it.DocID, it.ChunkIndex, it.Score, it.Text)
	}
	return strings.TrimRight(b.String(), "\n")
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

// DocumentConnectionCreated 建立文档编辑连接成功的自然语言响应
func DocumentConnectionCreated(documentID, botID string) string {
	return fmt.Sprintf("已建立与文档 %s 的编辑连接(bot_id: %s), 现在可以对其执行替换或插入。", documentID, botID)
}

// DocumentConnectionCreatedWithFeedback 建立连接成功的自然语言响应, 附带用户补充说明
func DocumentConnectionCreatedWithFeedback(documentID, botID, feedback string) string {
	msg := DocumentConnectionCreated(documentID, botID)
	if strings.TrimSpace(feedback) == "" {
		return msg
	}
	return fmt.Sprintf("%s\n用户补充说明：%s", msg, strings.TrimSpace(feedback))
}

// DocumentConnectionRejected 用户拒绝建立编辑连接时的自然语言响应, 作为工具结果回传模型
func DocumentConnectionRejected(documentID, feedback string) string {
	if strings.TrimSpace(feedback) == "" {
		return fmt.Sprintf("用户拒绝对文档 %s 的编辑授权，操作已取消。不要再次尝试建立连接。", documentID)
	}
	return fmt.Sprintf("用户拒绝对文档 %s 的编辑授权，操作已取消。用户补充说明：%s。不要再次尝试建立连接。",
		documentID, strings.TrimSpace(feedback))
}

// DocumentConnectionContent 机器人视角文档内容的自然语言响应
func DocumentConnectionContent(content string) string {
	if strings.TrimSpace(content) == "" {
		return "文档当前没有可返回的内容或变化。"
	}
	return content
}

// DocumentConnectionApplied 一次编辑提交成功的自然语言响应
func DocumentConnectionApplied(action string) string {
	return fmt.Sprintf("已%s。", action)
}
