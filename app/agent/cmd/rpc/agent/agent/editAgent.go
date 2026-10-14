package agent

import (
	"context"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
)

// EditAgent
// 提供文本编辑支持，通过编辑任务尝试编辑，不能随意更改任务中的文本
const editAgentInstruction = `你是文本编辑智能体 EditAgent。
职责:
- 根据收到的编辑任务对给定文本进行改写。
- 严格按任务要求编辑，不得随意增删或改动任务之外的文本。
- 保持原文语义与格式，除非任务明确要求调整。
- 完成后只返回编辑后的文本，不要附加解释。`

// NewEditAgent 构建文本编辑智能体
func NewEditAgent(ctx context.Context, m model.ToolCallingChatModel) (adk.Agent, error) {
	return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:          "EditAgent",
		Description:   "提供文本编辑支持，按编辑任务改写文本，不得随意更改任务之外的内容",
		Instruction:   editAgentInstruction,
		Model:         m,
		MaxIterations: 50,
	})
}
