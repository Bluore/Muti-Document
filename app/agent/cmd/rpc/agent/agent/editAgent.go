package agent

import (
	"context"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
)

// EditAgent
// 文档编辑智能体: 通过编辑连接对指定文档做替换/插入, 不能随意改动任务之外的文本
const editAgentInstruction = `你是文档编辑智能体 EditAgent。
职责:
- 根据收到的编辑任务, 对指定文档进行改写。
- 编辑前必须先调用 createDocumentConnection 建立与该文档的编辑连接, 同一文档只需建立一次。
- 不确定文档当前内容时, 先调用 getDocumentConnection 获取最新内容或变更差异, 再决定如何编辑。
- 用 replaceDocumentConnection 替换一段原文, 用 insertDocumentConnection 在锚点后插入新内容。
- 严格按任务要求编辑, 不得随意增删或改动任务之外的文本。
- 完成后简要说明做了哪些修改, 不要输出整篇文档。`

// NewEditAgent 构建文档编辑智能体, 挂载机器人编辑连接相关工具
func NewEditAgent(ctx context.Context, m model.ToolCallingChatModel, tools ...tool.BaseTool) (adk.Agent, error) {
	return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "EditAgent",
		Description: "对指定文档执行文本编辑(替换/插入), 按编辑任务改写, 不得随意更改任务之外的内容",
		Instruction: editAgentInstruction,
		Model:       m,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: tools,
			},
		},
		MaxIterations: 50,
	})
}
