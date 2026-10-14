package tool

import (
	"context"
	"errors"

	"github.com/Bluore/mutidoc/app/agent/cmd/rpc/agent/tool/toolcallresp"
	"github.com/Bluore/mutidoc/app/document/cmd/rpc/document"
	"github.com/Bluore/mutidoc/pkg/ctxdata"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

// ListDocumentToolName 查询当前用户创建的全部文档的工具名
const ListDocumentToolName = "listDocument"

// listPageSize 分页拉取时的每页数量, document-rpc 单页上限为 100
const listPageSize int64 = 100

// listDocumentReq 查询文档列表入参, 无需模型提供参数
type listDocumentReq struct{}

// NewListDocumentTool 构建查询当前用户创建的全部文档的工具。
// 用户身份取自会话上下文, 只列出该用户自己创建的文档。
// 工具错误经 toolcallresp 转成自然语言结果回传模型, 不中断整轮对话。
func NewListDocumentTool(documentRpc document.DocumentZrpcClient) (tool.InvokableTool, error) {
	it, err := utils.InferTool(ListDocumentToolName,
		"查询当前用户创建的全部文档, 返回每篇文档的 ID、标题与更新时间, 用于了解资料库里有哪些文档。",
		func(ctx context.Context, _ listDocumentReq) (string, error) {
			return listDocuments(ctx, documentRpc)
		})
	if err != nil {
		return nil, err
	}
	return utils.WrapInvokableToolWithErrorHandler(it, toolcallresp.ErrorHandler), nil
}

// listDocuments 分页拉取当前用户创建的全部文档
func listDocuments(ctx context.Context, documentRpc document.DocumentZrpcClient) (string, error) {
	if documentRpc == nil {
		return toolcallresp.NewDocumentListFailedErr(errors.New("document rpc client 未初始化"))
	}

	// 用户身份只信任会话上下文, 不采用模型传入的值
	userID, ok := ctxdata.GetUserIdFromCtx(ctx)
	if !ok || userID == "" {
		return toolcallresp.NewUserNotIdentifiedErr()
	}

	items := make([]toolcallresp.DocumentItem, 0)
	for page := int64(1); ; page++ {
		resp, err := documentRpc.GetDocumentList(ctx, &document.GetDocumentListReq{
			OwnerId: userID,
			Page:    page,
			Size:    listPageSize,
		})
		if err != nil {
			return toolcallresp.NewDocumentListFailedErr(err)
		}

		docs := resp.GetDocumentInfo()
		for _, doc := range docs {
			items = append(items, toolcallresp.DocumentItem{
				ID:       doc.GetId(),
				Summary:  doc.GetSummary(),
				UpdateAt: doc.GetUpdateAt(),
			})
		}

		// 该页为空或已取满总数则结束
		if len(docs) == 0 || int64(len(items)) >= resp.GetTotal() {
			break
		}
	}

	return toolcallresp.DocumentList(items), nil
}
