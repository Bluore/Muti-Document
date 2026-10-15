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

// InsertDocumentConnectionToolName 在文档锚点后插入内容的工具名
const InsertDocumentConnectionToolName = "insertDocumentConnection"

// insertDocumentConnectionReq 插入文档内容入参
type insertDocumentConnectionReq struct {
	DocumentID string `json:"document_id" jsonschema_description:"目标文档 ID"`
	Anchor     string `json:"anchor" jsonschema_description:"锚点原文, 新内容将插入到它后面; 传空字符串表示插入到文档开头"`
	Insertion  string `json:"insertion" jsonschema_description:"要插入的新文本"`
}

// NewInsertDocumentConnectionTool 构建在文档锚点后插入内容的工具。
// 在机器人本地快照中匹配锚点并在其后插入, 匹配不到时提示先读取最新内容。
// 工具错误经 toolcallresp 转成自然语言结果回传模型, 不中断整轮对话。
func NewInsertDocumentConnectionTool(editRpc edit.Edit) (tool.InvokableTool, error) {
	it, err := utils.InferTool(InsertDocumentConnectionToolName,
		"在文档中一段锚点原文的后面插入新的文本, 锚点必须与文档当前内容完全一致。",
		func(ctx context.Context, req insertDocumentConnectionReq) (string, error) {
			return insertDocumentConnection(ctx, editRpc, req)
		})
	if err != nil {
		return nil, err
	}
	return utils.WrapInvokableToolWithErrorHandler(it, toolcallresp.ErrorHandler), nil
}

// insertDocumentConnection 提交一次文本插入
func insertDocumentConnection(ctx context.Context, editRpc edit.Edit, req insertDocumentConnectionReq) (string, error) {
	if req.DocumentID == "" {
		return toolcallresp.NewDocumentIDRequiredErr()
	}
	if req.Insertion == "" {
		return toolcallresp.NewEditTextRequiredErr()
	}
	if editRpc == nil {
		return toolcallresp.NewDocumentConnectionFailedErr(errors.New("edit rpc client 未初始化"))
	}

	userID, ok := ctxdata.GetUserIdFromCtx(ctx)
	if !ok || userID == "" {
		return toolcallresp.NewUserNotIdentifiedErr()
	}

	if _, err := editRpc.Insert(ctx, &edit.InsertReq{
		BotId:     botIDFor(userID, req.DocumentID),
		Anchor:    req.Anchor,
		Insertion: req.Insertion,
	}); err != nil {
		return mapEditRpcErr(err)
	}

	return toolcallresp.DocumentConnectionApplied("完成内容插入"), nil
}
