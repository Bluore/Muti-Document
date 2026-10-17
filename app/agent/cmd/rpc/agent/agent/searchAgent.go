package agent

import (
	"context"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"

	"github.com/Bluore/mutidoc/app/agent/cmd/rpc/agent/prompt"
)

// NewSearchAgent 构建文档检索子智能体, 挂载文档查询工具
func NewSearchAgent(ctx context.Context, m model.ToolCallingChatModel, tools ...tool.BaseTool) (adk.Agent, error) {
	return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "SearchAgent",
		Description: "承担复杂的多步文档任务: 跨文档检索比对、内容总结、逐篇遍历及其他长任务; 单次简单搜索不必调用它",
		Instruction: prompt.SearchAgent,
		Model:       m,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: tools,
			},
		},
		MaxIterations: 50,
	})
}
