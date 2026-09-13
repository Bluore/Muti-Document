// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package login

import (
	"context"
	"fmt"
	"strings"

	"github.com/Bluore/mutidoc/app/user/cmd/api/internal/httperr"
	"github.com/Bluore/mutidoc/app/user/cmd/api/internal/svc"
	"github.com/Bluore/mutidoc/app/user/cmd/api/internal/types"
	"github.com/Bluore/mutidoc/app/user/cmd/rpc/user"
	"github.com/Bluore/mutidoc/pkg/globalkey"
	"github.com/Bluore/mutidoc/pkg/grpcerr"
	"github.com/Bluore/mutidoc/pkg/response"
	"google.golang.org/grpc/status"

	"github.com/zeromicro/go-zero/core/logx"
)

type RegisterLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// register
func NewRegisterLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RegisterLogic {
	return &RegisterLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RegisterLogic) Register(req *types.RegisterReq) (resp *types.RegisterResp, err error) {
	// email格式化
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	if ok := globalkey.RegexpEmail.MatchString(req.Email); !ok {
		return nil, httperr.ErrEmailFormatUnsupported
	}

	// 获取验证码
	verifyCode, err := l.svcCtx.RedisClient.Get(l.ctx, fmt.Sprintf(globalkey.RedisKeyUserVerifyKey, req.Email)).Result()
	if err != nil {
		logx.WithContext(l.ctx).Errorf("redis fail: %v", err)
		return nil, httperr.ErrEmailNotSendVerify
	}

	// verify
	if req.EmailCode != verifyCode {
		logx.WithContext(l.ctx).Errorw("verify code fail", logx.Field("user_code", req.EmailCode))
		return nil, httperr.ErrEmailVerifyFail
	}

	_, err = l.svcCtx.RedisClient.Del(l.ctx, fmt.Sprintf(globalkey.RedisKeyUserVerifyKey, req.Email)).Result()
	if err != nil {
		logx.WithContext(l.ctx).Errorf("redis fail: %v", err)
		return nil, response.ErrServer
	}

	_, err = l.svcCtx.RedisClient.Del(l.ctx, fmt.Sprintf(globalkey.RedisKeyUserVerifyLimit, req.Email)).Result()
	if err != nil {
		logx.WithContext(l.ctx).Errorf("redis fail: %v", err)
		return nil, response.ErrServer
	}

	// register
	userDetail, err := l.svcCtx.UserRpc.CreateUser(l.ctx, &user.CreateUserReq{
		NickName: req.NickName,
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		if st, ok := status.FromError(err); ok {
			if grpcerr.USER_NICKNAME_EXIST_ERROR.Equal(st) {
				return nil, httperr.ErrNicknameExist
			}
			if grpcerr.USER_EMAIL_EXIST_ERROR.Equal(st) {
				return nil, httperr.ErrEmailExist
			}
		}
		logx.WithContext(l.ctx).Errorf("create user fail: %v", err)
		return nil, response.ErrServer
	}
	logx.WithContext(l.ctx).Infow("register success", logx.Field("user_info", userDetail))

	// generate token
	token, err := l.svcCtx.UserRpc.GenerateToken(l.ctx, &user.GenerateTokenReq{
		UserId: userDetail.UserID,
	})
	if err != nil || token == nil {
		logx.WithContext(l.ctx).Errorf("generate token fail: %v", err)
		return nil, response.ErrServer
	}

	return &types.RegisterResp{
		AccessToken:  token.AccessToken,
		AccessExpire: token.AccessExpire,
	}, nil
}
