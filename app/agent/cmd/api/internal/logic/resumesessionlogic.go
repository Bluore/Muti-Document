// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package logic

import (
	"context"

	"github.com/Bluore/mutidoc/app/agent/cmd/api/internal/svc"
	"github.com/Bluore/mutidoc/app/agent/cmd/api/internal/types"

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
	// todo: 调用 agent-rpc ResumeSession 恢复会话
	return
}
