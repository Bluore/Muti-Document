package agent

import (
	"context"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"

	"github.com/Bluore/mutidoc/app/agent/cmd/rpc/agent/prompt"
)

// NewEditAgent 构建文档编辑智能体, 挂载机器人编辑连接相关工具
func NewEditAgent(ctx context.Context, m model.ToolCallingChatModel, tools ...tool.BaseTool) (adk.Agent, error) {
	return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "EditAgent",
		Description: "对指定文档执行文本编辑(替换/插入), 按编辑任务改写, 不得随意更改任务之外的内容",
		Instruction: prompt.EditAgent,
		Model:       m,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: tools,
			},
		},
		MaxIterations: 50,
	})
}
