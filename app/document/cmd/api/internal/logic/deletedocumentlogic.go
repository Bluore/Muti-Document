// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package logic

import (
	"context"

	"github.com/Bluore/mutidoc/app/document/cmd/api/internal/svc"
	"github.com/Bluore/mutidoc/app/document/cmd/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteDocumentLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// delete document
func NewDeleteDocumentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteDocumentLogic {
	return &DeleteDocumentLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteDocumentLogic) DeleteDocument(req *types.DeleteDocumentReq) (resp *types.DeleteDocumentResp, err error) {
	// todo: add your logic here and delete this line

	return
}
