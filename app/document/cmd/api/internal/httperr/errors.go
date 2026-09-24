package httperr

import "github.com/Bluore/mutidoc/pkg/response"

var (
	ErrUpgradeWebSocket     = response.NewError(200, 1024001, "升级为WebSocket请求失败")
	ErrDocumentNotExist     = response.NewError(200, 1024002, "文档不存在")
	ErrDocumentNoPermission = response.NewError(200, 1024003, "没有权限操作该文档")
)
