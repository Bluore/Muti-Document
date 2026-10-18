// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package logic

import (
	"context"

	"github.com/Bluore/mutidoc/app/agent/cmd/api/internal/httperr"
	"github.com/Bluore/mutidoc/app/agent/cmd/api/internal/svc"
	"github.com/Bluore/mutidoc/app/agent/cmd/api/internal/types"
	"github.com/Bluore/mutidoc/app/agent/cmd/rpc/agent"
	"github.com/Bluore/mutidoc/pkg/ctxdata"
	"github.com/Bluore/mutidoc/pkg/grpcerr"
	"github.com/Bluore/mutidoc/pkg/response"

	"github.com/zeromicro/go-zero/core/logx"
)

type HumanFeedbackLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// human feedback
func NewHumanFeedbackLogic(ctx context.Context, svcCtx *svc.ServiceContext) *HumanFeedbackLogic {
	return &HumanFeedbackLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// HumanFeedback 提交人工审批结果, 恢复被打断的会话
func (l *HumanFeedbackLogic) HumanFeedback(req *types.HumanFeedbackReq) (resp *types.HumanFeedbackResp, err error) {
	userID := ctxdata.MustGetUserIdFromCtx(l.ctx)

	_, err = l.svcCtx.AgentRpc.HumanFeedback(l.ctx, &agent.HumanFeedbackReq{
		SessionId: req.SessionID,
		UserId:    userID,
		Approved:  req.Approved,
		Feedback:  req.Feedback,
	})
	if err != nil {
		switch {
		case grpcerr.AGENT_SESSION_NOT_EXIST_ERROR.EqualError(err):
			return nil, httperr.ErrSessionNotExist
		case grpcerr.AGENT_SESSION_NOT_INTERRUPTED.EqualError(err):
			return nil, httperr.ErrNotInterrupted
		case grpcerr.AGENT_SESSION_BUSY_ERROR.EqualError(err):
			return nil, httperr.ErrSessionBusy
		}
		l.Errorf("human feedback fail: %v", err)
		return nil, response.ErrServer
	}

	return &types.HumanFeedbackResp{}, nil
}
