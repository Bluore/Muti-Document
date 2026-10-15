package agent

import (
	"context"

	"github.com/Bluore/mutidoc/app/agent/cmd/rpc/agent/botconn"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/zeromicro/go-zero/core/logx"
)

// QuitBotConnectionFunc 退出一个机器人在文档中的编辑连接 (edit-rpc)
type QuitBotConnectionFunc func(ctx context.Context, botID string) error

// autoQuitAgentTool 包装子智能体工具: 一次运行成功结束后, 自动退出本次建立的机器人连接
type autoQuitAgentTool struct {
	inner tool.InvokableTool
	quit  QuitBotConnectionFunc
}

// NewAutoQuitAgentTool 包装子智能体工具, 在其运行成功结束后自动退出本次建立的机器人连接。
// inner 不可直接调用或 quit 为空时原样返回。
func NewAutoQuitAgentTool(inner tool.BaseTool, quit QuitBotConnectionFunc) tool.BaseTool {
	it, ok := inner.(tool.InvokableTool)
	if !ok || quit == nil {
		return inner
	}
	return &autoQuitAgentTool{inner: it, quit: quit}
}

func (t *autoQuitAgentTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return t.inner.Info(ctx)
}

// InvokableRun 运行子智能体; 成功后退出其建立的机器人连接, 失败或被打断时保留连接以便恢复
func (t *autoQuitAgentTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	out, err := t.inner.InvokableRun(ctx, argumentsInJSON, opts...)
	if err != nil {
		return out, err
	}

	for _, botID := range botconn.Drain(ctx) {
		if quitErr := t.quit(ctx, botID); quitErr != nil {
			logx.WithContext(ctx).Errorw("auto quit bot connection fail",
				logx.Field("bot_id", botID),
				logx.Field("error", quitErr),
			)
		}
	}
	return out, nil
}
