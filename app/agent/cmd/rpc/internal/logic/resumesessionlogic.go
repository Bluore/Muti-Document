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

type ResumeSessionLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewResumeSessionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ResumeSessionLogic {
	return &ResumeSessionLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 恢复会话
func (l *ResumeSessionLogic) ResumeSession(in *pb.ResumeSessionReq) (*pb.ResumeSessionResp, error) {
	if in.SessionId == "" || in.UserId == "" || in.Query == "" {
		return nil, grpcerr.REQUEST_PARAM_ERROR
	}

	client, err := session.Get(l.ctx, l.svcCtx.Session, in.SessionId)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return nil, grpcerr.AGENT_SESSION_NOT_EXIST_ERROR
		}
		logx.WithContext(l.ctx).Errorf("get session fail: %v", err)
		return nil, errors.Wrapf(grpcerr.SERVER_COMMON_ERROR, "get session error: %v", err)
	}
	if client.GetSession().UserID != in.UserId {
		return nil, grpcerr.AGENT_SESSION_NOT_EXIST_ERROR
	}

	// 提交一轮对话给会话客户端, 后台生成并落库
	if err = client.Resume(in.Query); err != nil {
		if errors.Is(err, session.ErrGenerating) {
			return nil, grpcerr.AGENT_SESSION_BUSY_ERROR
		}
		logx.WithContext(l.ctx).Errorf("resume session fail: %v", err)
		return nil, errors.Wrapf(grpcerr.SERVER_COMMON_ERROR, "resume session error: %v", err)
	}

	return &pb.ResumeSessionResp{Session: toPbSession(client)}, nil
}
