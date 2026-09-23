package httperr

import "github.com/Bluore/mutidoc/pkg/response"

var (
	ErrUpgradeWebSocket = response.NewError(200, 1024001, "升级为WebSocket请求失败")
)
