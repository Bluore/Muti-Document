// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package grantLink

import (
	"context"

	"github.com/Bluore/mutidoc/app/link/cmd/api/internal/httperr"
	"github.com/Bluore/mutidoc/app/link/cmd/api/internal/svc"
	"github.com/Bluore/mutidoc/app/link/cmd/api/internal/types"
	"github.com/Bluore/mutidoc/app/link/cmd/rpc/link"
	"github.com/Bluore/mutidoc/pkg/ctxdata"
	"github.com/Bluore/mutidoc/pkg/grpcerr"
	"github.com/Bluore/mutidoc/pkg/response"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetGrantLinkLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// grant document permission and open link
func NewGetGrantLinkLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetGrantLinkLogic {
	return &GetGrantLinkLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetGrantLinkLogic) GetGrantLink(req *types.GetGrantDocumentReq) (resp *types.GetGrantDocumentResp, err error) {
	// 被授权用户从 jwt 中获取
	userID := ctxdata.MustGetUserIdFromCtx(l.ctx)
	if req.GrantID == "" {
		return nil, response.ErrValidation
	}

	// 授权链接会为当前用户授予文档权限, 并返回可跳转的链接
	result, err := l.svcCtx.LinkRpc.GetGrantLink(l.ctx, &link.GetGrantLinkReq{
		GrantId: req.GrantID,
		UserId:  userID,
	})
	if err != nil {
		if grpcerr.LINK_NOT_EXIST_ERROR.EqualError(err) {
			return nil, httperr.ErrLinkNotExist
		}
		if grpcerr.LINK_EXPIRED_ERROR.EqualError(err) {
			return nil, httperr.ErrLinkExpired
		}
		if grpcerr.LINK_NO_PERMISSION_ERROR.EqualError(err) {
			return nil, httperr.ErrLinkNoPermission
		}
		logx.WithContext(l.ctx).Errorf("get grant link fail: %v", err)
		return nil, response.ErrServer
	}

	return &types.GetGrantDocumentResp{
		Link: result.TargetUrl,
	}, nil
}
