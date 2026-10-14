package tool

import (
	"context"
	"errors"

	"github.com/Bluore/mutidoc/app/agent/cmd/rpc/agent/tool/toolcallresp"
	"github.com/Bluore/mutidoc/app/document/cmd/rpc/document"
	"github.com/Bluore/mutidoc/pkg/ctxdata"
	"github.com/Bluore/mutidoc/pkg/grpcerr"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

// QueryDocumentToolName 按文档 ID 查询完整内容的工具名
const QueryDocumentToolName = "queryDocument"

// queryDocumentReq 查询文档内容入参
type queryDocumentReq struct {
	DocumentID string `json:"document_id" jsonschema_description:"目标文档的 ID"`
}

// NewQueryDocumentTool 构建按文档 ID 查询完整内容的工具。
// 调用前会校验当前会话用户对文档的权限, 无权限时拒绝返回内容。
// 工具错误经 toolcallresp 转成自然语言结果回传模型, 不中断整轮对话。
func NewQueryDocumentTool(documentRpc document.DocumentZrpcClient) (tool.InvokableTool, error) {
	it, err := utils.InferTool(QueryDocumentToolName,
		"根据 document_id 查询指定文档的完整内容(标题与正文)。仅当当前用户拥有该文档权限时返回内容, 否则拒绝。",
		func(ctx context.Context, req queryDocumentReq) (string, error) {
			return queryDocument(ctx, documentRpc, req)
		})
	if err != nil {
		return nil, err
	}
	return utils.WrapInvokableToolWithErrorHandler(it, toolcallresp.ErrorHandler), nil
}

// queryDocument 校验权限后读取文档内容
func queryDocument(ctx context.Context, documentRpc document.DocumentZrpcClient, req queryDocumentReq) (string, error) {
	if req.DocumentID == "" {
		return toolcallresp.NewDocumentIDRequiredErr()
	}
	if documentRpc == nil {
		return toolcallresp.NewDocumentQueryFailedErr(errors.New("document rpc client 未初始化"))
	}

	// 用户身份只信任会话上下文, 不采用模型传入的值
	userID, ok := ctxdata.GetUserIdFromCtx(ctx)
	if !ok || userID == "" {
		return toolcallresp.NewUserNotIdentifiedErr()
	}

	// 先校验权限, 无权限直接拒绝
	permission, err := documentRpc.GetDocumentPermission(ctx, &document.GetDocumentPermissionReq{
		UserId:     userID,
		DocumentId: req.DocumentID,
	})
	if err != nil {
		return rpcErrResp(err)
	}
	if !permission.GetHasPermission() {
		return toolcallresp.NewDocumentNoPermissionErr()
	}

	content, err := documentRpc.GetDocumentContent(ctx, &document.GetDocumentContentReq{
		DocumentId: req.DocumentID,
	})
	if err != nil {
		return rpcErrResp(err)
	}

	return toolcallresp.DocumentContent(content.GetSummary(), content.GetContent(), content.GetVersion()), nil
}

// rpcErrResp 把 document-rpc 错误归类为工具错误响应
func rpcErrResp(err error) (string, error) {
	if grpcerr.DOCUMENT_NOT_EXIST_ERROR.EqualError(err) {
		return toolcallresp.NewDocumentNotExistErr()
	}
	return toolcallresp.NewDocumentQueryFailedErr(err)
}
