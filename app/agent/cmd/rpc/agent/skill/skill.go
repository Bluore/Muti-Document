package skill

import (
	"context"
	_ "embed"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

// supportSkillDoc 网站功能与使用文档, 编译期嵌入
//
//go:embed supportSkill.md
var supportSkillDoc string

// SupportSkillName 网站支持技能的对外工具名
const SupportSkillName = "loadSupportSkill"

// loadSupportSkillReq 加载技能的入参
type loadSupportSkillReq struct {
	Topic string `json:"topic,omitempty" jsonschema_description:"希望了解的功能或问题主题, 可留空返回完整文档"`
}

// NewSupportSkillTool 构建加载网站支持文档的工具, MainAgent 在回答网站相关问题时调用
func NewSupportSkillTool() (tool.InvokableTool, error) {
	return utils.InferTool(SupportSkillName,
		"加载本站(supportSkill)的功能与使用文档。当用户咨询网站功能、使用方法或常见问题时, 先调用本工具获取文档, 再依据文档作答。",
		func(_ context.Context, _ loadSupportSkillReq) (string, error) {
			return supportSkillDoc, nil
		})
}
