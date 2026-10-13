package httperr

import "github.com/Bluore/mutidoc/pkg/response"

var (
	ErrStreamUnsupported = response.NewError(200, 1035001, "当前连接不支持流式响应")
)
