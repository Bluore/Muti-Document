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

type ResumeSessionLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// resume session
func NewResumeSessionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ResumeSessionLogic {
	return &ResumeSessionLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ResumeSessionLogic) ResumeSession(req *types.ResumeSessionReq) (resp *types.ResumeSessionResp, err error) {
	userID := ctxdata.MustGetUserIdFromCtx(l.ctx)

	_, err = l.svcCtx.AgentRpc.ResumeSession(l.ctx, &agent.ResumeSessionReq{
		SessionId: req.SessionID,
		UserId:    userID,
		Query:     req.Query,
	})
	if err != nil {
		if grpcerr.AGENT_SESSION_NOT_EXIST_ERROR.EqualError(err) {
			return nil, httperr.ErrSessionNotExist
		}
		if grpcerr.AGENT_SESSION_BUSY_ERROR.EqualError(err) {
			return nil, httperr.ErrSessionBusy
		}
		l.Errorf("resume session fail: %v", err)
		return nil, response.ErrServer
	}

	return &types.ResumeSessionResp{}, nil
}
