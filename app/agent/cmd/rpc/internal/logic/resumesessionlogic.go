package logic

import (
	"context"

	"github.com/Bluore/mutidoc/app/agent/cmd/rpc/internal/svc"
	"github.com/Bluore/mutidoc/app/agent/cmd/rpc/pb"

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
	// todo: add your logic here and delete this line

	return &pb.ResumeSessionResp{}, nil
}
