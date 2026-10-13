package httperr

import "github.com/Bluore/mutidoc/pkg/response"

var (
	ErrUpgradeWebSocket = response.NewError(200, 1024001, "升级为WebSocket请求失败")
	ErrDocumentNotExist = response.NewError(200, 1024002, "文档不存在")
	ErrUserNotExist     = response.NewError(200, 1024004, "用户不存在")
)
