// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package logic

import (
	"context"

	"github.com/Bluore/mutidoc/app/agent/cmd/api/internal/svc"
	"github.com/Bluore/mutidoc/app/agent/cmd/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListSessionLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// list session
func NewListSessionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListSessionLogic {
	return &ListSessionLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListSessionLogic) ListSession(req *types.ListSessionReq) (resp *types.ListSessionResp, err error) {
	// todo: add your logic here and delete this line

	return
}
