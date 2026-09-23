// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package logic

import (
	"context"

	"github.com/Bluore/mutidoc/app/document/cmd/api/internal/svc"
	"github.com/zeromicro/go-zero/core/logx"
)

type InfoDocumentLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// get document info
func NewInfoDocumentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *InfoDocumentLogic {
	return &InfoDocumentLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *InfoDocumentLogic) InfoDocument() error {
	// todo: add your logic here and delete this line

	return nil
}
