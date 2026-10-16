package tool

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/Bluore/mutidoc/app/agent/cmd/rpc/agent/tool/toolcallresp"
	"github.com/Bluore/mutidoc/app/document/cmd/rpc/document"
	"github.com/Bluore/mutidoc/pkg/ctxdata"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

// SearchDocumentToolName 向量检索文档内容的工具名
const SearchDocumentToolName = "searchDocument"

// searchDocumentDefaultTopK 默认返回条数, 与 document-rpc 的缺省值一致
const searchDocumentDefaultTopK int64 = 3

// searchDocumentMaxTopK 单次返回条数上限, 避免模型一次拉回过多正文
const searchDocumentMaxTopK int64 = 20

// searchDocumentReq 向量检索入参
type searchDocumentReq struct {
	Query      string `json:"query" jsonschema_description:"要检索的内容, 用自然语言描述或关键词"`
	DocumentID string `json:"document_id" jsonschema_description:"可选, 只在该文档内检索; 留空则检索当前用户的全部文档"`
	TopK       int64  `json:"top_k" jsonschema_description:"可选, 返回条数, 默认 3, 最多 20"`
}

// searchDocumentChunk 命中分块, 字段与 document-rpc 返回的 JSON 对齐
type searchDocumentChunk struct {
	DocID      string  `json:"doc_id"`
	ChunkIndex int64   `json:"chunk_index"`
	Text       string  `json:"text"`
	Score      float32 `json:"score"`
}

// NewSearchDocumentTool 构建向量检索文档内容的工具。
// 检索范围限定当前用户: 不指定文档时只查其创建的文档, 指定文档时先校验权限。
// 工具错误经 toolcallresp 转成自然语言结果回传模型, 不中断整轮对话。
func NewSearchDocumentTool(documentRpc document.DocumentZrpcClient) (tool.InvokableTool, error) {
	it, err := utils.InferTool(SearchDocumentToolName,
		"按语义检索文档内容, 返回与检索内容最相似的若干条原文分块(含文档 ID、分块序号与相似度)。适合查找零散的相关内容; 需要整篇文档时改用 queryDocument。",
		func(ctx context.Context, req searchDocumentReq) (string, error) {
			return searchDocument(ctx, documentRpc, req)
		})
	if err != nil {
		return nil, err
	}
	return utils.WrapInvokableToolWithErrorHandler(it, toolcallresp.ErrorHandler), nil
}

// searchDocument 校验用户与权限后做向量检索
func searchDocument(ctx context.Context, documentRpc document.DocumentZrpcClient, req searchDocumentReq) (string, error) {
	query := strings.TrimSpace(req.Query)
	if query == "" {
		return toolcallresp.NewSearchQueryRequiredErr()
	}
	if documentRpc == nil {
		return toolcallresp.NewDocumentSearchFailedErr(errors.New("document rpc client 未初始化"))
	}

	// 用户身份只信任会话上下文, 不采用模型传入的值
	userID, ok := ctxdata.GetUserIdFromCtx(ctx)
	if !ok || userID == "" {
		return toolcallresp.NewUserNotIdentifiedErr()
	}

	// 限定了文档就先校验权限, 无权限直接拒绝
	if req.DocumentID != "" {
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
	}

	resp, err := documentRpc.SearchDocument(ctx, &document.SearchDocumentReq{
		Text:       query,
		TopK:       clampTopK(req.TopK),
		DocumentId: req.DocumentID,
		OwnerId:    ownerFilter(req.DocumentID, userID),
	})
	if err != nil {
		return toolcallresp.NewDocumentSearchFailedErr(err)
	}

	var chunks []searchDocumentChunk
	if err := json.Unmarshal([]byte(resp.GetResult()), &chunks); err != nil {
		return toolcallresp.NewDocumentSearchFailedErr(err)
	}

	items := make([]toolcallresp.DocumentChunkItem, 0, len(chunks))
	for _, c := range chunks {
		items = append(items, toolcallresp.DocumentChunkItem{
			DocID:      c.DocID,
			ChunkIndex: c.ChunkIndex,
			Text:       c.Text,
			Score:      c.Score,
		})
	}

	return toolcallresp.DocumentSearchHits(items), nil
}

// clampTopK 把返回条数收敛到 [默认值, 上限]
func clampTopK(topK int64) int64 {
	if topK <= 0 {
		return searchDocumentDefaultTopK
	}
	if topK > searchDocumentMaxTopK {
		return searchDocumentMaxTopK
	}
	return topK
}

// ownerFilter 未限定文档时按归属过滤, 保证只检索当前用户自己的文档;
// 限定了文档时权限已校验, 不再按归属过滤(可能是别人共享给该用户的文档)
func ownerFilter(documentID, userID string) string {
	if documentID != "" {
		return ""
	}
	return userID
}
