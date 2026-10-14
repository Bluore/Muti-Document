package logic

import (
	"context"

	"github.com/Bluore/mutidoc/app/agent/cmd/rpc/agent/session"
	"github.com/Bluore/mutidoc/app/agent/cmd/rpc/internal/svc"
	"github.com/Bluore/mutidoc/app/agent/cmd/rpc/pb"
	"github.com/Bluore/mutidoc/app/agent/model"
	"github.com/Bluore/mutidoc/pkg/grpcerr"
	"github.com/pkg/errors"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetSessionLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetSessionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetSessionLogic {
	return &GetSessionLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 获取会话
func (l *GetSessionLogic) GetSession(in *pb.GetSessionReq) (*pb.GetSessionResp, error) {
	if in.SessionId == "" {
		return nil, grpcerr.REQUEST_PARAM_ERROR
	}

	// 会话客户端不存在时自动读库重建
	client, err := session.Get(l.ctx, l.svcCtx.Session, in.SessionId)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return nil, grpcerr.AGENT_SESSION_NOT_EXIST_ERROR
		}
		logx.WithContext(l.ctx).Errorf("get session fail: %v", err)
		return nil, errors.Wrapf(grpcerr.SERVER_COMMON_ERROR, "get session error: %v", err)
	}

	// 从 offset 处返回增量内容 (History 内部维护字符前缀和)
	data, newOffset := client.HistorySSE(in.Offset)

	return &pb.GetSessionResp{
		Data:      string(data),
		NewOffset: newOffset,
		Status:    int32(client.Status()),
	}, nil
}
