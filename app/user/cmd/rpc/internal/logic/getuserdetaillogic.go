package logic

import (
	"context"

	"github.com/Bluore/mutidoc/app/user/cmd/rpc/internal/svc"
	"github.com/Bluore/mutidoc/app/user/cmd/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetUserDetailLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetUserDetailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserDetailLogic {
	return &GetUserDetailLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 获取用户
func (l *GetUserDetailLogic) GetUserDetail(in *pb.GetUserDetailReq) (*pb.GetUserDetailResp, error) {
	// todo: add your logic here and delete this line

	return &pb.GetUserDetailResp{}, nil
}
