// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package info

import (
	"context"

	"github.com/Bluore/mutidoc/app/user/cmd/api/internal/httperr"
	"github.com/Bluore/mutidoc/app/user/cmd/api/internal/svc"
	"github.com/Bluore/mutidoc/app/user/cmd/api/internal/types"
	"github.com/Bluore/mutidoc/app/user/cmd/rpc/user"
	"github.com/Bluore/mutidoc/pkg/ctxdata"
	"github.com/Bluore/mutidoc/pkg/grpcerr"
	"github.com/Bluore/mutidoc/pkg/response"

	"github.com/zeromicro/go-zero/core/logx"
)

type DetailLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// get user info
func NewDetailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DetailLogic {
	return &DetailLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DetailLogic) Detail(req *types.UserInfoReq) (resp *types.UserInfoResp, err error) {
	userID, ok := ctxdata.GetUserIdFromCtx(l.ctx)
	if !ok {
		return nil, response.ErrServer
	}
	result, err := l.svcCtx.UserRpc.GetUserDetail(l.ctx, &user.GetUserDetailReq{
		UserID: userID,
	})
	if err != nil {
		if grpcerr.USER_NOT_EXIST_ERROR.EqualError(err) {
			return nil, httperr.ErrUserNotExist
		}
		return nil, response.ErrServer
	}

	return &types.UserInfoResp{
		UserInfo: types.User{
			Id:       result.UserID,
			NickName: result.NickName,
			Email:    result.Email,
			Avatar:   result.Avatar,
		},
	}, nil
}
