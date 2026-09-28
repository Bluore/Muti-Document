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

type GetGrantLinkLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetGrantLinkLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetGrantLinkLogic {
	return &GetGrantLinkLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 获取授予用户文档权限对应文档
func (l *GetGrantLinkLogic) GetGrantLink(in *pb.GetGrantDocumentReq) (*pb.GetGrantDocumentResp, error) {
	if in.GrantId == "" {
		return nil, grpcerr.REQUEST_PARAM_ERROR
	}

	grantLink, err := l.svcCtx.GrantLinkModel.FindOne(l.ctx, in.GrantId)
	if err != nil && err != model.ErrNotFound {
		logx.WithContext(l.ctx).Errorf("FindOne grant link fail: %v", err)
		return nil, errors.Wrapf(grpcerr.SERVER_COMMON_ERROR, "db query error: %v", err)
	}
	if grantLink == nil {
		return nil, grpcerr.LINK_NOT_EXIST_ERROR
	}
	if grantLink.ExpireAt.Before(time.Now()) {
		return nil, grpcerr.LINK_EXPIRED_ERROR
	}

	// 仅返回文档与所有者, 授权与跳转地址由 api 层处理
	return &pb.GetGrantDocumentResp{
		Document: grantLink.DocumentId,
		OwnerId:  grantLink.UserId,
	}, nil
}
