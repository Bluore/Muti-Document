package logic

import (
	"context"
	"time"

	"github.com/Bluore/mutidoc/app/link/cmd/rpc/internal/svc"
	"github.com/Bluore/mutidoc/app/link/cmd/rpc/pb"
	"github.com/Bluore/mutidoc/app/link/model"
	"github.com/Bluore/mutidoc/pkg/grpcerr"
	"github.com/Bluore/mutidoc/pkg/idgenerate"
	"github.com/pkg/errors"

	"github.com/zeromicro/go-zero/core/logx"
)

const (
	// grantLinkPathPrefix 授权链接访问路径, 与 link-api 路由保持一致
	grantLinkPathPrefix = "/grant/"
	// defaultGrantLinkExpire 未指定过期时间时的默认有效期
	defaultGrantLinkExpire = 7 * 24 * time.Hour
)

type CreateGrantLinkLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateGrantLinkLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateGrantLinkLogic {
	return &CreateGrantLinkLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 创建文档授权链接
func (l *CreateGrantLinkLogic) CreateGrantLink(in *pb.CreateGrantLinkReq) (*pb.CreateGrantLinkResp, error) {
	if in.DocumentId == "" || in.UserId == "" {
		return nil, grpcerr.REQUEST_PARAM_ERROR
	}

	// 同一文档与用户已存在授权链接时直接复用
	exist, err := l.svcCtx.GrantLinkModel.FindOneByDocumentIdAndUserId(l.ctx, in.DocumentId, in.UserId)
	if err != nil && err != model.ErrNotFound {
		logx.WithContext(l.ctx).Errorf("FindOneByDocumentIdAndUserId fail: %v", err)
		return nil, errors.Wrapf(grpcerr.SERVER_COMMON_ERROR, "db query error: %v", err)
	}
	if exist != nil {
		return l.buildResp(exist.GrantId), nil
	}

	expireAt := time.Unix(in.ExpireAt, 0)
	if in.ExpireAt <= 0 {
		expireAt = time.Now().Add(defaultGrantLinkExpire)
	}

	grantId := idgenerate.NewID(l.svcCtx.Config.ServerID)
	_, err = l.svcCtx.GrantLinkModel.Insert(l.ctx, nil, &model.GrantDocumentLinks{
		GrantId:    grantId,
		DocumentId: in.DocumentId,
		UserId:     in.UserId,
		ExpireAt:   expireAt,
		Version:    1,
	})
	if err != nil {
		logx.WithContext(l.ctx).Errorf("insert grant link fail: %v", err)
		return nil, errors.Wrapf(grpcerr.SERVER_COMMON_ERROR, "db insert error: %v", err)
	}
	logx.WithContext(l.ctx).Infow("create grant link success",
		logx.Field("grant_id", grantId),
		logx.Field("document_id", in.DocumentId),
		logx.Field("user_id", in.UserId),
	)

	return l.buildResp(grantId), nil
}

// buildResp 拼接授权链接访问地址
func (l *CreateGrantLinkLogic) buildResp(grantId string) *pb.CreateGrantLinkResp {
	return &pb.CreateGrantLinkResp{
		GrantLink: l.svcCtx.Config.BaseUrl + grantLinkPathPrefix + grantId,
	}
}
