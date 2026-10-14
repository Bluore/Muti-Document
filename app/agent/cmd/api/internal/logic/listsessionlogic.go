// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package logic

import (
	"context"

	"github.com/Bluore/mutidoc/app/agent/cmd/api/internal/svc"
	"github.com/Bluore/mutidoc/app/agent/cmd/api/internal/types"
	"github.com/Bluore/mutidoc/app/agent/cmd/rpc/agent"
	"github.com/Bluore/mutidoc/pkg/ctxdata"
	"github.com/Bluore/mutidoc/pkg/response"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListSessionLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// list session
func NewListSessionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListSessionLogic {
	return &ListSessionLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListSessionLogic) ListSession(req *types.ListSessionReq) (resp *types.ListSessionResp, err error) {
	// 会话归属者从 jwt 中获取
	userID := ctxdata.MustGetUserIdFromCtx(l.ctx)

	result, err := l.svcCtx.AgentRpc.ListSession(l.ctx, &agent.ListSessionReq{UserId: userID})
	if err != nil {
		l.Errorf("list session fail: %v", err)
		return nil, response.ErrServer
	}

	sessions := make([]types.Session, 0, len(result.GetSessions()))
	for _, s := range result.GetSessions() {
		if s == nil {
			continue
		}
		sessions = append(sessions, types.Session{
			SessionID:    s.GetSessionId(),
			SessionTitle: s.GetSessionTitle(),
			UserID:       s.GetUserId(),
		})
	}

	return &types.ListSessionResp{
		Sessions:   sessions,
		SessionLen: len(sessions),
	}, nil
}
