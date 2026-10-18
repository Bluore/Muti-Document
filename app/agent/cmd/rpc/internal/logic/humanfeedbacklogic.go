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

type HumanFeedbackLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewHumanFeedbackLogic(ctx context.Context, svcCtx *svc.ServiceContext) *HumanFeedbackLogic {
	return &HumanFeedbackLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 人工反馈, 恢复被打断的会话
func (l *HumanFeedbackLogic) HumanFeedback(in *pb.HumanFeedbackReq) (*pb.HumanFeedbackResp, error) {
	if in.SessionId == "" || in.UserId == "" {
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

	if err = client.HumanFeedback(in.Approved, in.Feedback); err != nil {
		switch {
		case errors.Is(err, session.ErrNotInterrupted):
			return nil, grpcerr.AGENT_SESSION_NOT_INTERRUPTED
		case errors.Is(err, session.ErrGenerating):
			return nil, grpcerr.AGENT_SESSION_BUSY_ERROR
		}
		logx.WithContext(l.ctx).Errorf("human feedback fail: %v", err)
		return nil, errors.Wrapf(grpcerr.SERVER_COMMON_ERROR, "human feedback error: %v", err)
	}

	return &pb.HumanFeedbackResp{Session: toPbSession(client)}, nil
}
