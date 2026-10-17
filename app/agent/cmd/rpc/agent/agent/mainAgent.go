package agent

import (
	"context"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"

	"github.com/Bluore/mutidoc/app/agent/cmd/rpc/agent/llmmodel"
	"github.com/Bluore/mutidoc/app/agent/cmd/rpc/agent/prompt"
	"github.com/Bluore/mutidoc/app/agent/cmd/rpc/agent/skill"
)

// Config 顶层智能体配置
type Config struct {
	Model llmmodel.DeepSeekConf
	// DocumentTools 挂载到 SearchAgent 的文档工具(供复杂检索/遍历使用)
	DocumentTools []tool.BaseTool
	// EditTools 挂载到 EditAgent 的文档编辑工具(建立连接/替换/插入/读取)
	EditTools []tool.BaseTool
	// ExtraTools 额外挂载到 MainAgent 的工具(普通文档检索/读全文、MCP 工具等)
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
		Name:        "MuDo",
		Description: "统一入口智能体，处理文档查询/编辑与网站功能咨询",
		Instruction: prompt.MainAgent,
		Model:       m,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: tools,
			},
		},
		MaxIterations: 100,
	})
}
