package tool

import (
	"fmt"

	"github.com/Bluore/mutidoc/app/agent/cmd/rpc/agent/tool/toolcallresp"
	"github.com/Bluore/mutidoc/pkg/grpcerr"
)

// botIDFor 由用户与文档确定机器人在文档中的编辑身份,
// 保证同一用户对同一文档的多次调用命中同一个 BotConnection。
func botIDFor(userID, documentID string) string {
	return fmt.Sprintf("agent-bot-%s-%s", userID, documentID)
}

// mapEditRpcErr 把 edit-rpc 错误映射为自然语言工具错误
func mapEditRpcErr(err error) (string, error) {
	switch {
	case grpcerr.DOCUMENT_NOT_EXIST_ERROR.EqualError(err):
		return toolcallresp.NewDocumentNotExistErr()
	case grpcerr.BOT_NOT_EXIST_ERROR.EqualError(err):
		return toolcallresp.NewBotConnectionNotExistErr()
	case grpcerr.BOT_TEXT_NOT_FOUND_ERROR.EqualError(err):
		return toolcallresp.NewBotTextNotFoundErr()
	default:
		return toolcallresp.NewDocumentConnectionFailedErr(err)
	}
}
