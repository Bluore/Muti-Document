package logic

import (
	"context"

	"github.com/Bluore/mutidoc/app/link/cmd/rpc/internal/svc"
	"github.com/Bluore/mutidoc/app/link/cmd/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateGrantLinkLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateGrantLinkLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateGrantLinkLogic {
	return &CreateGrantLinkLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 创建文档授权链接
func (l *CreateGrantLinkLogic) CreateGrantLink(in *pb.CreateGrantLinkReq) (*pb.CreateGrantLinkResp, error) {
	// todo: add your logic here and delete this line

	return &pb.CreateGrantLinkResp{}, nil
}
