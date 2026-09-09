package logic

import (
	"context"

	"github.com/Bluore/mutidoc/app/user/cmd/rpc/internal/svc"
	"github.com/Bluore/mutidoc/app/user/cmd/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateUserLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateUserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateUserLogic {
	return &CreateUserLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 创建用户
func (l *CreateUserLogic) CreateUser(in *pb.CreateUserReq) (*pb.CreateUserResp, error) {
	// todo: add your logic here and delete this line

	return &pb.CreateUserResp{}, nil
}
