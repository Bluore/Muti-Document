package logic

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/Bluore/mutidoc/app/document/cmd/rpc/internal/svc"
	"github.com/Bluore/mutidoc/app/document/cmd/rpc/pb"
	"github.com/Bluore/mutidoc/app/document/model"
	"github.com/Bluore/mutidoc/pkg/grpcerr"
	"github.com/pkg/errors"

	"github.com/zeromicro/go-zero/core/logx"
)

type SearchDocumentLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSearchDocumentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SearchDocumentLogic {
	return &SearchDocumentLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 向量检索文档内容(返回相似度最高的前 K 条分块)
func (l *SearchDocumentLogic) SearchDocument(in *pb.SearchDocumentReq) (*pb.SearchDocumentResp, error) {
	text := strings.TrimSpace(in.Text)
	if text == "" {
		return nil, grpcerr.REQUEST_PARAM_ERROR
	}
	if l.svcCtx.DocumentChunks == nil {
		return nil, errors.Wrapf(grpcerr.SERVER_COMMON_ERROR, "vector store not configured")
	}

	vector, err := l.svcCtx.Embedder.EmbedOne(l.ctx, text)
	if err != nil {
		logx.WithContext(l.ctx).Errorf("embed query fail: %v", err)
		return nil, errors.Wrapf(grpcerr.SERVER_COMMON_ERROR, "embed query error: %v", err)
	}

	chunks, err := l.svcCtx.DocumentChunks.Search(l.ctx, vector, int(in.TopK), model.DocumentChunkFilter{
		DocID:   in.DocumentId,
		OwnerID: in.OwnerId,
	})
	if err != nil {
		logx.WithContext(l.ctx).Errorf("search document chunks fail: %v", err)
		return nil, errors.Wrapf(grpcerr.SERVER_COMMON_ERROR, "search document chunks error: %v", err)
	}

	result, err := json.Marshal(chunks)
	if err != nil {
		logx.WithContext(l.ctx).Errorf("marshal search result fail: %v", err)
		return nil, errors.Wrapf(grpcerr.SERVER_COMMON_ERROR, "marshal search result error: %v", err)
	}

	return &pb.SearchDocumentResp{Result: string(result)}, nil
}
