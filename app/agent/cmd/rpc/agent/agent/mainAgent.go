package agent

import (
	"context"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"

	"github.com/Bluore/mutidoc/app/agent/cmd/rpc/agent/llmmodel"
	"github.com/Bluore/mutidoc/app/agent/cmd/rpc/agent/skill"
)

// MainAgent
// 统一入口智能体: 自主识别任务类型, 文档类任务调用 searchAgent/editAgent, 网站咨询加载 supportSkill
const mainAgentInstruction = `你是智能体 MainAgent，是本站的唯一入口智能体。
职责:
- 先理解用户意图, 再决定直接回答还是调用工具, 完成后整合结果回复用户。
- 文档任务: 需要检索文档内容时调用 searchAgent; 需要编辑某篇文档时调用 editAgent, 并在任务中说明文档 ID 与编辑要求。
- 网站支持: 回答关于本站功能、使用方式与常见问题的咨询; 作答前先调用 loadSupportSkill 加载网站文档, 并严格依据文档内容回答。
- 已挂载的外部工具(如高德地图相关工具)按各自描述在合适场景下调用。
- 一次只调用一个子智能体, 不要并行调用。
- 超出能力范围或信息不足时明确说明, 不要编造。`

// Config 顶层智能体配置
type Config struct {
	Model llmmodel.DeepSeekConf
	// DocumentTools 挂载到 SearchAgent 的文档工具(如按 ID 查询文档内容)
	DocumentTools []tool.BaseTool
	// EditTools 挂载到 EditAgent 的文档编辑工具(建立连接/替换/插入/读取)
	EditTools []tool.BaseTool
	// ExtraTools 额外挂载到 MainAgent 的工具(如 MCP 工具)
	ExtraTools []tool.BaseTool
	// QuitBotConnection 退出机器人编辑连接, EditAgent 结束后由程序自动调用
	QuitBotConnection QuitBotConnectionFunc
}

// NewAgent 组装完整智能体: 创建对话模型并构建 MainAgent 入口
func NewAgent(ctx context.Context, conf Config) (adk.Agent, error) {
	m, err := llmmodel.NewDeepSeekModel(ctx, conf.Model)
	if err != nil {
		return nil, err
	}

	return NewMainAgent(ctx, m, conf.DocumentTools, conf.EditTools, conf.ExtraTools, conf.QuitBotConnection)
}

// NewMainAgent 构建统一入口智能体, 以 searchAgent/editAgent 作为文档工具, 并挂载 extraTools
func NewMainAgent(ctx context.Context, m model.ToolCallingChatModel, documentTools, editTools, extraTools []tool.BaseTool, quitBot QuitBotConnectionFunc) (adk.Agent, error) {
	searchAgent, err := NewSearchAgent(ctx, m, documentTools...)
	if err != nil {
		return nil, err
	}

	editAgent, err := NewEditAgent(ctx, m, editTools...)
	if err != nil {
		return nil, err
	}

	supportSkill, err := skill.NewSupportSkillTool()
	if err != nil {
		return nil, err
	}

	// EditAgent 结束后自动退出其建立的机器人编辑连接
	editAgentTool := NewAutoQuitAgentTool(adk.NewAgentTool(ctx, editAgent), quitBot)

	tools := []tool.BaseTool{
		adk.NewAgentTool(ctx, searchAgent),
		editAgentTool,
		supportSkill,
	}
	tools = append(tools, extraTools...)

	return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "MainAgent",
		Description: "统一入口智能体，处理文档查询/编辑与网站功能咨询",
		Instruction: mainAgentInstruction,
		Model:       m,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: tools,
			},
		},
		MaxIterations: 100,
	})
}
