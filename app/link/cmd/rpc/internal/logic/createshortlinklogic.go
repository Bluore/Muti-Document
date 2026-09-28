package logic

import (
	"context"
	"time"

	"github.com/Bluore/mutidoc/app/link/cmd/rpc/internal/svc"
	"github.com/Bluore/mutidoc/app/link/cmd/rpc/pb"
	"github.com/Bluore/mutidoc/app/link/model"
	"github.com/Bluore/mutidoc/pkg/base62"
	"github.com/Bluore/mutidoc/pkg/globalkey"
	"github.com/Bluore/mutidoc/pkg/grpcerr"
	"github.com/pkg/errors"
	"github.com/redis/go-redis/v9"

	"github.com/zeromicro/go-zero/core/logx"
)

const (
	// shortLinkPathPrefix 短链公开访问路径, 与 link-api 路由保持一致
	shortLinkPathPrefix = "/s/"
	// shortLinkIDOffset 叠加在自增ID上的常数, 避免短链长度过短
	shortLinkIDOffset uint64 = 1000000
	// defaultShortLinkExpire 未指定过期时间时的默认有效期
	defaultShortLinkExpire = 7 * 24 * time.Hour
)

type CreateShortLinkLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateShortLinkLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateShortLinkLogic {
	return &CreateShortLinkLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 创建短链
func (l *CreateShortLinkLogic) CreateShortLink(in *pb.CreateShortLinkReq) (*pb.CreateShortLinkResp, error) {
	if in.TargetUrl == "" {
		return nil, grpcerr.REQUEST_PARAM_ERROR
	}

	// 目标地址已存在短链时直接复用
	exist, err := l.svcCtx.ShortLinksModel.FindOneByTargetUrl(l.ctx, in.TargetUrl)
	if err != nil && err != model.ErrNotFound {
		logx.WithContext(l.ctx).Errorf("FindOneByTargetUrl fail: %v", err)
		return nil, errors.Wrapf(grpcerr.SERVER_COMMON_ERROR, "db query error: %v", err)
	}
	if exist != nil {
		return l.buildResp(exist.LinkId), nil
	}

	linkId, err := l.generateLinkID()
	if err != nil {
		return nil, err
	}

	expireAt := time.Unix(in.ExpireAt, 0)
	if in.ExpireAt <= 0 {
		expireAt = time.Now().Add(defaultShortLinkExpire)
	}

	_, err = l.svcCtx.ShortLinksModel.Insert(l.ctx, nil, &model.ShortLinks{
		LinkId:    linkId,
		TargetUrl: in.TargetUrl,
		ExpireAt:  expireAt,
		Version:   1,
	})
	if err != nil {
		logx.WithContext(l.ctx).Errorf("insert short link fail: %v", err)
		return nil, errors.Wrapf(grpcerr.SERVER_COMMON_ERROR, "db insert error: %v", err)
	}
	logx.WithContext(l.ctx).Infow("create short link success",
		logx.Field("link_id", linkId),
		logx.Field("target_url", in.TargetUrl),
	)

	return l.buildResp(linkId), nil
}

// generateLinkID 通过 redis 事务自增获取全局唯一ID, 再叠加常数做 base62 编码
func (l *CreateShortLinkLogic) generateLinkID() (string, error) {
	var incr *redis.IntCmd
	_, err := l.svcCtx.RedisClient.TxPipelined(l.ctx, func(pipe redis.Pipeliner) error {
		incr = pipe.Incr(l.ctx, globalkey.RedisKeyShortLinkId)
		return nil
	})
	if err != nil {
		logx.WithContext(l.ctx).Errorf("incr short link id fail: %v", err)
		return "", errors.Wrapf(grpcerr.REDIS_ERROR, "redis incr error: %v", err)
	}

	return base62.Encode(uint64(incr.Val()) + shortLinkIDOffset), nil
}

// buildResp 拼接短链访问地址
func (l *CreateShortLinkLogic) buildResp(linkId string) *pb.CreateShortLinkResp {
	return &pb.CreateShortLinkResp{
		ShortLink: l.svcCtx.Config.BaseUrl + shortLinkPathPrefix + linkId,
	}
}
