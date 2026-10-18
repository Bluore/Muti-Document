package logic

import (
	"context"
	"encoding/json"

	"github.com/Bluore/mutidoc/app/agent/cmd/rpc/agent"
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

	// 将前端上下文转为自然语言, 前置到本轮提问送入模型
	query := in.Query
	if cond := parseCondition(in.Condition); cond != nil {
		if natural := cond.ToNatural(); len(natural) > 0 {
			query = string(natural) + "\n\n" + in.Query
		}
	}

	// 提交一轮对话给会话客户端, 后台生成并落库
	if err = client.Generate(query); err != nil {
		if errors.Is(err, session.ErrGenerating) || errors.Is(err, session.ErrInterrupted) {
			return nil, grpcerr.AGENT_SESSION_BUSY_ERROR
		}
		logx.WithContext(l.ctx).Errorf("resume session fail: %v", err)
		return nil, errors.Wrapf(grpcerr.SERVER_COMMON_ERROR, "resume session error: %v", err)
	}

	return &pb.ResumeSessionResp{Session: toPbSession(client)}, nil
}

// parseCondition 解析前端传入的上下文条件, 格式非法时忽略
func parseCondition(raw string) *agent.ResumeCondition {
	if raw == "" {
		return nil
	}
	cond := &agent.ResumeCondition{}
	if err := json.Unmarshal([]byte(raw), cond); err != nil {
		logx.Errorf("parse resume condition fail: %v", err)
		return nil
	}
	return cond
}
