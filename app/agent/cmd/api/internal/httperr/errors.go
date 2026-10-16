package httperr

import "github.com/Bluore/mutidoc/pkg/response"

var (
	ErrStreamUnsupported = response.NewError(200, 1035001, "当前连接不支持流式响应")
	ErrSessionNotExist   = response.NewError(200, 1044002, "会话不存在")
	ErrSessionBusy       = response.NewError(200, 1044003, "会话正在生成中")
	ErrConditionInvalid  = response.NewError(200, 1044004, "上下文条件格式错误")
)
