package agent

import (
	"context"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
)

// SearchAgent
// 承担复杂的多步文档任务(跨文档检索、总结、遍历), 只返回原文或摘要, 不做改写
const searchAgentInstruction = `你是文档检索智能体 SearchAgent, 负责复杂、需要多步执行的文档任务。
职责:
- 复杂检索: 需要跨多篇文档比对, 或要按多个条件反复检索才能定位内容时, 综合使用 searchDocument 与 listDocument。
- 总结: 汇总多篇文档或大段原文的要点。
- 遍历: 逐篇浏览用户资料库, 用 listDocument 枚举后逐篇 queryDocument, 或分段 searchDocument。
- 其他长任务: 需要多轮工具调用才能完成的任务。
- 只返回检索到的原文或摘要，不做改写、不编造内容; 未找到结果时明确说明。
- 单次、目标明确的普通搜索不由你处理, 上层 MainAgent 会直接调用工具。`

// NewSearchAgent 构建文档检索子智能体, 挂载文档查询工具
func NewSearchAgent(ctx context.Context, m model.ToolCallingChatModel, tools ...tool.BaseTool) (adk.Agent, error) {
	return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "SearchAgent",
		Description: "承担复杂的多步文档任务: 跨文档检索比对、内容总结、逐篇遍历及其他长任务; 单次简单搜索不必调用它",
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
