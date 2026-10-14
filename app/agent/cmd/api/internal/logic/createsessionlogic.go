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

type CreateSessionLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// create session
func NewCreateSessionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateSessionLogic {
	return &CreateSessionLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateSessionLogic) CreateSession(req *types.CreateSessionReq) (resp *types.CreateSessionResp, err error) {
	// 会话归属者从 jwt 中获取
	userID := ctxdata.MustGetUserIdFromCtx(l.ctx)

	result, err := l.svcCtx.AgentRpc.CreateSession(l.ctx, &agent.CreateSessionReq{
		UserId: userID,
		Title:  req.Query,
	})
	if err != nil {
		l.Errorf("create session fail: %v", err)
		return nil, response.ErrServer
	}
	sessionID := result.GetSession().GetSessionId()

	// 创建后立即提交首轮对话, 触发 agent 实际生成
	if _, err = l.svcCtx.AgentRpc.ResumeSession(l.ctx, &agent.ResumeSessionReq{
		SessionId: sessionID,
		UserId:    userID,
		Query:     req.Query,
	}); err != nil {
		if grpcerr.AGENT_SESSION_BUSY_ERROR.EqualError(err) {
			return nil, httperr.ErrSessionBusy
		}
		l.Errorf("resume session fail: %v", err)
		return nil, response.ErrServer
	}

	l.Infow("create session success", logx.Field("session_id", sessionID))

	return &types.CreateSessionResp{
		SessionID: sessionID,
	}, nil
}
