// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package login

import (
	"context"
	"strings"

	"github.com/Bluore/mutidoc/app/user/cmd/api/internal/httperr"
	"github.com/Bluore/mutidoc/app/user/cmd/api/internal/svc"
	"github.com/Bluore/mutidoc/app/user/cmd/api/internal/types"
	"github.com/Bluore/mutidoc/app/user/cmd/rpc/user"
	"github.com/Bluore/mutidoc/pkg/globalkey"
	"github.com/Bluore/mutidoc/pkg/grpcerr"
	"github.com/Bluore/mutidoc/pkg/response"
	"github.com/zeromicro/go-zero/core/logx"
)

type LoginLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// login
func NewLoginLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LoginLogic {
	return &LoginLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *LoginLogic) Login(req *types.LoginReq) (resp *types.LoginResp, err error) {
	// email格式化
	req.Password = strings.ToLower(strings.TrimSpace(req.Password))
	if ok := globalkey.RegexpEmail.MatchString(req.Password); !ok {
		return nil, httperr.ErrEmailFormatUnsupported
	}

	// todo 验证robot

	// login user
	result, err := l.svcCtx.UserRpc.LoginUser(l.ctx, &user.LoginUserReq{
		Login:    req.Login,
		Password: req.Password,
	})
	if err != nil {
		if grpcerr.USER_PASSWORD_ERROR.EqualError(err) {
			return nil, httperr.ErrPasswordWrong
		}
		if grpcerr.USER_NOT_EXIST_ERROR.EqualError(err) {
			return nil, httperr.ErrUserNotExist
		}
		return nil, response.ErrServer
	}

	return &types.LoginResp{
		AccessToken:  result.AccessToken,
		AccessExpire: result.AccessExpire,
	}, nil
}
