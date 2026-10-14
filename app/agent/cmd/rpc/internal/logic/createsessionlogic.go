package logic

import (
	"context"

	"github.com/Bluore/mutidoc/app/agent/cmd/rpc/agent/session"
	"github.com/Bluore/mutidoc/app/agent/cmd/rpc/internal/svc"
	"github.com/Bluore/mutidoc/app/agent/cmd/rpc/pb"
	"github.com/Bluore/mutidoc/pkg/grpcerr"
	"github.com/pkg/errors"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateSessionLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateSessionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateSessionLogic {
	return &CreateSessionLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 创建会话
func (l *CreateSessionLogic) CreateSession(in *pb.CreateSessionReq) (*pb.CreateSessionResp, error) {
	if in.UserId == "" {
		return nil, grpcerr.REQUEST_PARAM_ERROR
	}

	// 由会话客户端负责建会话并落库
	client, err := session.Create(l.ctx, l.svcCtx.Session, in.UserId, in.Title)
	if err != nil {
		logx.WithContext(l.ctx).Errorf("create session fail: %v", err)
		return nil, errors.Wrapf(grpcerr.SERVER_COMMON_ERROR, "create session error: %v", err)
	}
	logx.WithContext(l.ctx).Infow("create session success",
		logx.Field("session_id", client.SessionID),
		logx.Field("user_id", client.UserID),
	)

	return &pb.CreateSessionResp{Session: toPbSession(client)}, nil
}
