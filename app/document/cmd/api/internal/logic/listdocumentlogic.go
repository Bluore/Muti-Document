// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package logic

import (
	"context"

	"github.com/Bluore/mutidoc/app/document/cmd/api/internal/svc"
	"github.com/Bluore/mutidoc/app/document/cmd/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListDocumentLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// list document
func NewListDocumentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListDocumentLogic {
	return &ListDocumentLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListDocumentLogic) ListDocument(req *types.GetDocumentListReq) (resp *types.GetDocumentListResp, err error) {
	// todo: add your logic here and delete this line

	return
}
