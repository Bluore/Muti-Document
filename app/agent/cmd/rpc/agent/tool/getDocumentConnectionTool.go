package tool

import (
	"context"
	"errors"

	"github.com/Bluore/mutidoc/app/agent/cmd/rpc/agent/tool/toolcallresp"
	"github.com/Bluore/mutidoc/app/document/cmd/edit/edit"
	"github.com/Bluore/mutidoc/pkg/ctxdata"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

// GetDocumentConnectionToolName 读取机器人视角文档内容的工具名
const GetDocumentConnectionToolName = "getDocumentConnection"

// getDocumentConnectionReq 读取文档内容入参
type getDocumentConnectionReq struct {
	DocumentID string `json:"document_id" jsonschema_description:"目标文档 ID"`
}

// NewGetDocumentConnectionTool 构建读取机器人视角文档内容的工具。
// 首次返回文档全文, 之后返回自上次读取以来的变更差异(含被删除的文本)。
// 工具错误经 toolcallresp 转成自然语言结果回传模型, 不中断整轮对话。
func NewGetDocumentConnectionTool(editRpc edit.Edit) (tool.InvokableTool, error) {
	it, err := utils.InferTool(GetDocumentConnectionToolName,
		"读取机器人对指定文档的最新视角: 首次返回全文, 之后返回自上次读取以来的变更差异。编辑前用它与确认当前内容。",
		func(ctx context.Context, req getDocumentConnectionReq) (string, error) {
			return getDocumentConnection(ctx, editRpc, req)
		})
	if err != nil {
		return nil, err
	}
	return utils.WrapInvokableToolWithErrorHandler(it, toolcallresp.ErrorHandler), nil
}

// getDocumentConnection 读取机器人视角下的文档内容或差异
func getDocumentConnection(ctx context.Context, editRpc edit.Edit, req getDocumentConnectionReq) (string, error) {
	if req.DocumentID == "" {
		return toolcallresp.NewDocumentIDRequiredErr()
	}
	if editRpc == nil {
		return toolcallresp.NewDocumentConnectionFailedErr(errors.New("edit rpc client 未初始化"))
	}

	userID, ok := ctxdata.GetUserIdFromCtx(ctx)
	if !ok || userID == "" {
		return toolcallresp.NewUserNotIdentifiedErr()
	}

	resp, err := editRpc.Get(ctx, &edit.GetReq{BotId: botIDFor(userID, req.DocumentID)})
	if err != nil {
		return mapEditRpcErr(err)
	}

	return toolcallresp.DocumentConnectionContent(resp.GetContent()), nil
}
