package logic

import (
	"context"

	"github.com/Bluore/mutidoc/app/link/cmd/rpc/internal/svc"
	"github.com/Bluore/mutidoc/app/link/cmd/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetGrantLinkLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetGrantLinkLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetGrantLinkLogic {
	return &GetGrantLinkLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 授予用户文档权限并获取跳转链接
func (l *GetGrantLinkLogic) GetGrantLink(in *pb.GetGrantLinkReq) (*pb.GetGrantLinkResp, error) {
	// todo: add your logic here and delete this line

	return &pb.GetGrantLinkResp{}, nil
}
