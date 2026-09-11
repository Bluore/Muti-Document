package logic

import (
	"context"

	"github.com/Bluore/mutidoc/app/user/cmd/rpc/internal/svc"
	"github.com/Bluore/mutidoc/app/user/cmd/rpc/pb"
	"github.com/Bluore/mutidoc/app/user/model"
	"github.com/Bluore/mutidoc/pkg/grpcerr"
	"github.com/Bluore/mutidoc/pkg/idgenerate"
	"github.com/pkg/errors"

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
	userModel := &model.Users{
		Id:       idgenerate.NewID(l.svcCtx.Config.ServerID),
		NickName: in.NickName,
		Email:    in.Email,
	}
	logx.Infof("create user: %+v", userModel)

	userExist, err := l.svcCtx.UserModel.FindOne(l.ctx, userModel.Id)
	if err != nil && err != model.ErrNotFound {
		return nil, errors.Wrapf(grpcerr.SERVER_COMMON_ERROR, "db query error: %v", err)
	}
	if userExist != nil {
		return nil, errors.Wrapf(grpcerr.SERVER_COMMON_ERROR, "userModel.Id %s already exists", userModel.Id)
	}

	userExist, err = l.svcCtx.UserModel.FindOneByEmail(l.ctx, in.Email)
	if err != nil && err != model.ErrNotFound {
		return nil, errors.Wrapf(grpcerr.SERVER_COMMON_ERROR, "db query error: %v", err)
	}
	if userExist != nil {
		return nil, errors.Wrapf(grpcerr.USER_EMAIL_EXIST_ERROR, "userModel.Email %s already exists", in.Email)
	}

	userExist, err = l.svcCtx.UserModel.FindOneByNickName(l.ctx, in.NickName)
	if err != nil && err != model.ErrNotFound {
		return nil, errors.Wrapf(grpcerr.SERVER_COMMON_ERROR, "db query error: %v", err)
	}
	if userExist != nil {
		return nil, errors.Wrapf(grpcerr.USER_NICKNAME_EXIST_ERROR, "userModel.NickName %s already exists", in.NickName)
	}

	_, err = l.svcCtx.UserModel.Insert(l.ctx, nil, userModel)
	if err != nil {
		return nil, errors.Wrapf(grpcerr.SERVER_COMMON_ERROR, "db query error: %v", err)
	}

	userResp, err := l.svcCtx.UserModel.FindOne(l.ctx, userModel.Id)
	if err != nil {
		if err == model.ErrNotFound {
			return nil, errors.Wrapf(grpcerr.SERVER_COMMON_ERROR, "call back to db error: userModel not found")
		}
		return nil, errors.Wrapf(grpcerr.SERVER_COMMON_ERROR, "db query error: %v", err)
	}

	logx.Infof("create userResp: %+v", userResp)

	return &pb.CreateUserResp{
		UserID:   userResp.Id,
		NickName: userResp.NickName,
		Email:    userResp.Email,
		Avatar:   userResp.Avatar,
		CreateAt: userResp.CreateAt.Unix(),
		UpdateAt: userResp.UpdateAt.Unix(),
		DeleteAt: userResp.DeleteAt.Unix(),
	}, nil
}
