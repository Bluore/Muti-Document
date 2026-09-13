package logic

import (
	"context"
	"database/sql"

	"github.com/Bluore/mutidoc/app/user/cmd/rpc/internal/svc"
	"github.com/Bluore/mutidoc/app/user/cmd/rpc/pb"
	"github.com/Bluore/mutidoc/pkg/grpcerr"
	"github.com/Bluore/mutidoc/pkg/pwdhash"

	"github.com/zeromicro/go-zero/core/logx"
)

type LoginUserLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewLoginUserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LoginUserLogic {
	return &LoginUserLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 用户登录
func (l *LoginUserLogic) LoginUser(in *pb.LoginUserReq) (*pb.LoginUserResp, error) {
	generateTokenLogic := NewGenerateTokenLogic(l.ctx, l.svcCtx)

	// verify email
	userModel, err := l.svcCtx.UserModel.FindOneByEmail(l.ctx, in.Login)
	if err != nil && err != sql.ErrNoRows {
		logx.WithContext(l.ctx).Errorf("FindOneByEmail err: %v", err)
		return nil, grpcerr.SERVER_COMMON_ERROR
	}
	if userModel != nil {
		if ok := pwdhash.CompareHashPassword(in.Password, userModel.Password); !ok {
			return nil, grpcerr.USER_PASSWORD_ERROR
		}

		token, err := generateTokenLogic.GenerateToken(&pb.GenerateTokenReq{
			UserId: userModel.Id,
		})
		if err != nil {
			logx.WithContext(l.ctx).Errorf("GenerateToken err: %v", err)
			return nil, grpcerr.SERVER_COMMON_ERROR
		}
		return &pb.LoginUserResp{
			AccessToken:  token.AccessToken,
			AccessExpire: token.AccessExpire,
		}, nil
	}

	// verify nickname
	userModel, err = l.svcCtx.UserModel.FindOneByNickName(l.ctx, in.Login)
	if err != nil && err != sql.ErrNoRows {
		logx.WithContext(l.ctx).Errorf("FindOneByEmail err: %v", err)
		return nil, grpcerr.SERVER_COMMON_ERROR
	}
	if userModel != nil {
		if ok := pwdhash.CompareHashPassword(in.Password, userModel.Password); !ok {
			return nil, grpcerr.USER_PASSWORD_ERROR
		}

		token, err := generateTokenLogic.GenerateToken(&pb.GenerateTokenReq{
			UserId: userModel.Id,
		})
		if err != nil {
			logx.WithContext(l.ctx).Errorf("GenerateToken err: %v", err)
			return nil, grpcerr.SERVER_COMMON_ERROR
		}
		return &pb.LoginUserResp{
			AccessToken:  token.AccessToken,
			AccessExpire: token.AccessExpire,
		}, nil
	}

	return nil, grpcerr.USER_NOT_EXIST_ERROR
}
