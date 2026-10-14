package agent

import (
	"context"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
)

// SearchAgent
// 提供文档检索支持, 只返回原文或摘要, 不做改写
const searchAgentInstruction = `你是文档检索智能体 SearchAgent。
职责:
- 在用户资料库中检索与查询相关的文档内容。
- 需要了解用户有哪些文档时, 调用 listDocument 列出其创建的全部文档。
- 需要读取某篇文档的完整内容时, 调用 queryDocument。
- 只返回检索到的原文或摘要，不做改写、不编造内容。
- 未找到结果时明确说明。`

// NewSearchAgent 构建文档检索子智能体, 挂载文档查询工具
func NewSearchAgent(ctx context.Context, m model.ToolCallingChatModel, tools ...tool.BaseTool) (adk.Agent, error) {
	return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "SearchAgent",
		Description: "在用户资料库中检索文档内容",
		Instruction: searchAgentInstruction,
		Model:       m,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: tools,
			},
		},
		MaxIterations: 50,
	})
}
