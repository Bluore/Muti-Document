package tool

import (
	"context"
	"errors"

	"github.com/Bluore/mutidoc/app/agent/cmd/rpc/agent/approval"
	"github.com/Bluore/mutidoc/app/agent/cmd/rpc/agent/botconn"
	"github.com/Bluore/mutidoc/app/agent/cmd/rpc/agent/tool/toolcallresp"
	"github.com/Bluore/mutidoc/app/document/cmd/edit/edit"
	"github.com/Bluore/mutidoc/pkg/ctxdata"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

// CreateDocumentConnectionToolName 为文档建立机器人编辑连接的工具名
const CreateDocumentConnectionToolName = "createDocumentConnection"

// createDocumentConnectionReq 建立文档编辑连接入参
type createDocumentConnectionReq struct {
	DocumentID string `json:"document_id" jsonschema_description:"要建立协作编辑连接的文档 ID"`
	BotName    string `json:"bot_name" jsonschema_description:"机器人显示在协作者列表中的名称, 可留空"`
}

// NewCreateDocumentConnectionTool 构建建立文档编辑连接的工具。
// 同一用户对同一文档只需建立一次, 之后才能执行替换/插入/读取。
// 首次为某个文档建立连接时会打断并请求用户授权, 授权通过后本次会话不再重复询问。
// 工具错误经 toolcallresp 转成自然语言结果回传模型, 不中断整轮对话。
func NewCreateDocumentConnectionTool(editRpc edit.Edit) (tool.InvokableTool, error) {
	it, err := utils.InferTool(CreateDocumentConnectionToolName,
		"为当前用户与指定文档建立一个机器人协作编辑连接。编辑文档前必须先调用它。同一用户对同一文档只需建立一次。首次建立时会请求用户授权。",
		func(ctx context.Context, req createDocumentConnectionReq) (string, error) {
			return createDocumentConnection(ctx, editRpc, req)
		})
	if err != nil {
		return nil, err
	}
	return utils.WrapInvokableToolWithErrorHandler(it, toolcallresp.ErrorHandler), nil
}

// createDocumentConnection 以当前用户身份在指定文档上创建一个机器人连接
func createDocumentConnection(ctx context.Context, editRpc edit.Edit, req createDocumentConnectionReq) (string, error) {
	if req.DocumentID == "" {
		return toolcallresp.NewDocumentIDRequiredErr()
	}
	if editRpc == nil {
		return toolcallresp.NewDocumentConnectionFailedErr(errors.New("edit rpc client 未初始化"))
	}

	// 用户身份只信任会话上下文, 不采用模型传入的值
	userID, ok := ctxdata.GetUserIdFromCtx(ctx)
	if !ok || userID == "" {
		return toolcallresp.NewUserNotIdentifiedErr()
	}

	botName := req.BotName
	if botName == "" {
		botName = "AI 编辑助手"
	}

	// 首次为某个文档建立连接时先请求用户授权
	approved, feedback, err := requestApproval(ctx, req.DocumentID, botName)
	if err != nil {
		return "", err
	}
	if !approved {
		return toolcallresp.DocumentConnectionRejected(req.DocumentID, feedback), nil
	}

	botID := botIDFor(userID, req.DocumentID)

	if _, err = editRpc.CreateBot(ctx, &edit.CreateBotReq{
		DocumentId: req.DocumentID,
		BotId:      botID,
		BotName:    botName,
		UserId:     userID,
	}); err != nil {
		return mapEditRpcErr(err)
	}

	// 记录本次建立的连接, EditAgent 结束后由程序自动退出
	botconn.Track(ctx, botID)

	return toolcallresp.DocumentConnectionCreatedWithFeedback(req.DocumentID, botID, feedback), nil
}

// requestApproval 请求用户授权建立编辑连接; 已授权时直接放行。
// 返回 (是否批准, 用户补充说明, 错误)。未批准且 err 为 nil 表示用户明确拒绝。
func requestApproval(ctx context.Context, documentID, botName string) (bool, string, error) {
	if approval.Allowed(ctx, documentID) {
		return true, "", nil
	}

	info := approval.NewInfo(documentID, botName)
	wasInterrupted, hasState, state := tool.GetInterruptState[approval.Info](ctx)
	if wasInterrupted && hasState {
		info = state
	}
	// 恢复入参与打断时不一致时重新确认, 避免把授权用到别的文档
	if info.DocumentID != "" && info.DocumentID != documentID {
		info = approval.NewInfo(documentID, botName)
		return false, "", tool.StatefulInterrupt(ctx, info, info)
	}
	if !wasInterrupted || !hasState {
		return false, "", tool.StatefulInterrupt(ctx, info, info)
	}

	// 只处理明确以本打断点为目标的恢复; 其他场景保持打断状态
	isTarget, hasData, decision := tool.GetResumeContext[*approval.Decision](ctx)
	if !isTarget {
		return false, "", tool.StatefulInterrupt(ctx, info, info)
	}
	if !hasData || decision == nil || !decision.Approved {
		feedback := ""
		if hasData && decision != nil {
			feedback = decision.Feedback
		}
		return false, feedback, nil
	}

	// 本次会话内该文档不再重复询问
	approval.Allow(ctx, info.DocumentID)
	return true, decision.Feedback, nil
}
