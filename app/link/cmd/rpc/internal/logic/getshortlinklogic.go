package logic

import (
	"context"

	"github.com/Bluore/mutidoc/app/link/cmd/rpc/internal/svc"
	"github.com/Bluore/mutidoc/app/link/cmd/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetShortLinkLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetShortLinkLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetShortLinkLogic {
	return &GetShortLinkLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 打开短链, 获取跳转链接
func (l *GetShortLinkLogic) GetShortLink(in *pb.GetShortLinkReq) (*pb.GetShortLinkResp, error) {
	// todo: add your logic here and delete this line

	return &pb.GetShortLinkResp{}, nil
}
