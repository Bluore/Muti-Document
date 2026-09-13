package logic

import (
	"context"

	"github.com/Bluore/mutidoc/app/user/cmd/rpc/internal/svc"
	"github.com/Bluore/mutidoc/app/user/cmd/rpc/pb"
	"github.com/Bluore/mutidoc/app/user/model"
	"github.com/Bluore/mutidoc/pkg/grpcerr"

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
	userModel, err := l.svcCtx.UserModel.FindOne(l.ctx, in.UserID)
	if err != nil && err != model.ErrNotFound {
		return nil, grpcerr.SERVER_COMMON_ERROR
	}
	if userModel == nil {
		return nil, grpcerr.USER_NOT_EXIST_ERROR
	}

	return &pb.GetUserDetailResp{
		UserID:   userModel.Id,
		NickName: userModel.NickName,
		Email:    userModel.Email,
		Avatar:   userModel.Avatar,
		CreateAt: userModel.CreateAt.Unix(),
		UpdateAt: userModel.UpdateAt.Unix(),
		DeleteAt: userModel.DeleteAt.Unix(),
	}, nil
}
