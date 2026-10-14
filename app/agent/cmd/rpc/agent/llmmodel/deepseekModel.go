package llmmodel

import (
	"context"

	"github.com/cloudwego/eino-ext/components/model/deepseek"
	"github.com/cloudwego/eino/components/model"
)

// DeepSeekConf DeepSeek 对话模型配置
type DeepSeekConf struct {
	APIKey      string
	BaseURL     string
	Model       string
	Temperature float32
	MaxTokens   int
}

// NewDeepSeekModel 创建支持工具调用的 DeepSeek 对话模型
func NewDeepSeekModel(ctx context.Context, conf DeepSeekConf) (model.ToolCallingChatModel, error) {
	return deepseek.NewChatModel(ctx, &deepseek.ChatModelConfig{
		APIKey:      conf.APIKey,
		BaseURL:     conf.BaseURL,
		Model:       conf.Model,
		Temperature: conf.Temperature,
		MaxTokens:   conf.MaxTokens,
	})
}
