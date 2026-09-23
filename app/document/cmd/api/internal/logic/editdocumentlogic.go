// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package logic

import (
	"context"

	"github.com/Bluore/mutidoc/app/document/cmd/api/internal/svc"
	"github.com/zeromicro/go-zero/core/logx"
)

type EditDocumentLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// edit document
func NewEditDocumentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *EditDocumentLogic {
	return &EditDocumentLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *EditDocumentLogic) EditDocument() error {
	// todo: add your logic here and delete this line

	return nil
}
