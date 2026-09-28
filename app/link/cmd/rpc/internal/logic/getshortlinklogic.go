package logic

import (
	"context"
	"time"

	"github.com/Bluore/mutidoc/app/link/cmd/rpc/internal/svc"
	"github.com/Bluore/mutidoc/app/link/cmd/rpc/pb"
	"github.com/Bluore/mutidoc/app/link/model"
	"github.com/Bluore/mutidoc/pkg/grpcerr"
	"github.com/pkg/errors"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetShortLinkLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetShortLinkLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetShortLinkLogic {
	return &GetShortLinkLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 打开短链, 获取跳转链接
func (l *GetShortLinkLogic) GetShortLink(in *pb.GetShortLinkReq) (*pb.GetShortLinkResp, error) {
	if in.Id == "" {
		return nil, grpcerr.REQUEST_PARAM_ERROR
	}

	shortLink, err := l.svcCtx.ShortLinksModel.FindOne(l.ctx, in.Id)
	if err != nil && err != model.ErrNotFound {
		logx.WithContext(l.ctx).Errorf("FindOne short link fail: %v", err)
		return nil, errors.Wrapf(grpcerr.SERVER_COMMON_ERROR, "db query error: %v", err)
	}
	if shortLink == nil {
		return nil, grpcerr.LINK_NOT_EXIST_ERROR
	}
	if shortLink.ExpireAt.Before(time.Now()) {
		return nil, grpcerr.LINK_EXPIRED_ERROR
	}

	return &pb.GetShortLinkResp{
		TargetUrl: shortLink.TargetUrl,
	}, nil
}
