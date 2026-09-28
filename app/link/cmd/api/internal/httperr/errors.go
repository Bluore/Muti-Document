package httperr

import "github.com/Bluore/mutidoc/pkg/response"

var (
	ErrLinkNotExist     = response.NewError(200, 1034001, "链接不存在")
	ErrLinkExpired      = response.NewError(200, 1034002, "链接已过期")
	ErrLinkNoPermission = response.NewError(200, 1034003, "没有权限访问该链接")
)
