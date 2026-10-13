// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package logic

import (
	"context"

	"github.com/Bluore/mutidoc/app/agent/cmd/api/internal/svc"
	"github.com/Bluore/mutidoc/app/agent/cmd/api/internal/types"

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

func (l *HumanFeedbackLogic) HumanFeedback(req *types.HumanFeedbackReq) (resp *types.HumanFeedbackResp, err error) {
	// todo: add your logic here and delete this line

	return
}
