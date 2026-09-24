// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package logic

import (
	"context"

	"github.com/Bluore/mutidoc/app/document/cmd/api/internal/svc"
	"github.com/Bluore/mutidoc/app/document/cmd/api/internal/types"
	"github.com/Bluore/mutidoc/app/document/cmd/rpc/document"
	"github.com/Bluore/mutidoc/pkg/ctxdata"
	"github.com/Bluore/mutidoc/pkg/response"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateDocumentLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// create document
func NewCreateDocumentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateDocumentLogic {
	return &CreateDocumentLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateDocumentLogic) CreateDocument(req *types.CreateDocumentReq) (resp *types.CreateDocumentResp, err error) {
	// 文档归属者从jwt中获取
	ownerID := ctxdata.MustGetUserIdFromCtx(l.ctx)

	result, err := l.svcCtx.DocumentRpc.CreateDocument(l.ctx, &document.CreateDocumentReq{
		OwnerId: ownerID,
	})
	if err != nil {
		logx.WithContext(l.ctx).Errorf("create document fail: %v", err)
		return nil, response.ErrServer
	}
	logx.WithContext(l.ctx).Infow("create document success", logx.Field("document_id", result.DocumentId))

	return &types.CreateDocumentResp{
		DocumentId: result.DocumentId,
	}, nil
}
