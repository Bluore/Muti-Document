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

// ReplaceDocumentConnectionToolName 替换文档文本的工具名
const ReplaceDocumentConnectionToolName = "replaceDocumentConnection"

// replaceDocumentConnectionReq 替换文档文本入参
type replaceDocumentConnectionReq struct {
	DocumentID  string `json:"document_id" jsonschema_description:"目标文档 ID"`
	Original    string `json:"original" jsonschema_description:"文档中要被替换的原文, 必须与当前内容完全一致"`
	Replacement string `json:"replacement" jsonschema_description:"替换后的新文本, 传空字符串表示删除该处原文"`
}

// NewReplaceDocumentConnectionTool 构建替换文档文本的工具。
// 在机器人本地快照中匹配原文并替换, 匹配不到时提示先读取最新内容。
// 工具错误经 toolcallresp 转成自然语言结果回传模型, 不中断整轮对话。
func NewReplaceDocumentConnectionTool(editRpc edit.Edit) (tool.InvokableTool, error) {
	it, err := utils.InferTool(ReplaceDocumentConnectionToolName,
		"将文档中一段原文替换为新的文本, 原文必须与文档当前内容完全一致。删除文本时把替换文本留空。",
		func(ctx context.Context, req replaceDocumentConnectionReq) (string, error) {
			return replaceDocumentConnection(ctx, editRpc, req)
		})
	if err != nil {
		return nil, err
	}
	return utils.WrapInvokableToolWithErrorHandler(it, toolcallresp.ErrorHandler), nil
}

// replaceDocumentConnection 提交一次文本替换
func replaceDocumentConnection(ctx context.Context, editRpc edit.Edit, req replaceDocumentConnectionReq) (string, error) {
	if req.DocumentID == "" {
		return toolcallresp.NewDocumentIDRequiredErr()
	}
	if req.Original == "" {
		return toolcallresp.NewEditTextRequiredErr()
	}
	if editRpc == nil {
		return toolcallresp.NewDocumentConnectionFailedErr(errors.New("edit rpc client 未初始化"))
	}

	userID, ok := ctxdata.GetUserIdFromCtx(ctx)
	if !ok || userID == "" {
		return toolcallresp.NewUserNotIdentifiedErr()
	}

	if _, err := editRpc.Replace(ctx, &edit.ReplaceReq{
		BotId:       botIDFor(userID, req.DocumentID),
		Original:    req.Original,
		Replacement: req.Replacement,
	}); err != nil {
		return mapEditRpcErr(err)
	}

	return toolcallresp.DocumentConnectionApplied("完成文本替换"), nil
}
