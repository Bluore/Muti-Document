package logic

import (
	"context"

	"github.com/Bluore/mutidoc/app/link/cmd/rpc/internal/svc"
	"github.com/Bluore/mutidoc/app/link/cmd/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateShortLinkLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateShortLinkLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateShortLinkLogic {
	return &CreateShortLinkLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 创建短链
func (l *CreateShortLinkLogic) CreateShortLink(in *pb.CreateShortLinkReq) (*pb.CreateShortLinkResp, error) {
	// todo: add your logic here and delete this line

	return &pb.CreateShortLinkResp{}, nil
}
