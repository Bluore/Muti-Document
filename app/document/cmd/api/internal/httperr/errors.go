package httperr

import "github.com/Bluore/mutidoc/pkg/response"

var (
	ErrDocumentNotExist     = response.NewError(200, 1024002, "文档不存在")
	ErrDocumentNoPermission = response.NewError(200, 1024003, "没有权限操作该文档")
)
