// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package logic

import (
	"context"

	"github.com/Bluore/mutidoc/app/link/cmd/api/internal/httperr"
	"github.com/Bluore/mutidoc/app/link/cmd/api/internal/svc"
	"github.com/Bluore/mutidoc/app/link/cmd/api/internal/types"
	"github.com/Bluore/mutidoc/app/link/cmd/rpc/link"
	"github.com/Bluore/mutidoc/pkg/grpcerr"
	"github.com/Bluore/mutidoc/pkg/response"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetShortLinkLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// open short link
func NewGetShortLinkLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetShortLinkLogic {
	return &GetShortLinkLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetShortLinkLogic) GetShortLink(req *types.GetShortLinkReq) (resp *types.GetShortLinkResp, err error) {
	if req.ID == "" {
		return nil, response.ErrValidation
	}

	// 短链为公开访问, 无需登录; 由 rpc 解析短链并返回跳转地址
	result, err := l.svcCtx.LinkRpc.GetShortLink(l.ctx, &link.GetShortLinkReq{
		Id: req.ID,
	})
	if err != nil {
		if grpcerr.LINK_NOT_EXIST_ERROR.EqualError(err) {
			return nil, httperr.ErrLinkNotExist
		}
		if grpcerr.LINK_EXPIRED_ERROR.EqualError(err) {
			return nil, httperr.ErrLinkExpired
		}
		logx.WithContext(l.ctx).Errorf("get short link fail: %v", err)
		return nil, response.ErrServer
	}

	return &types.GetShortLinkResp{
		Link: result.TargetUrl,
	}, nil
}
