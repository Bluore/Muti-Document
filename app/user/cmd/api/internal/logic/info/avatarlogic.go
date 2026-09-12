// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package info

import (
	"context"

	"github.com/Bluore/mutidoc/app/user/cmd/api/internal/svc"
	"github.com/Bluore/mutidoc/app/user/cmd/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type AvatarLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// change user avatar
func NewAvatarLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AvatarLogic {
	return &AvatarLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AvatarLogic) Avatar(req *types.UploadAvatarReq) (resp *types.UploadAvatarResp, err error) {
	// todo: add your logic here and delete this line

	return
}
