package logic

import (
	"context"
	"encoding/json"

	"github.com/Bluore/mutidoc/app/agent/cmd/api/internal/httperr"
	"github.com/Bluore/mutidoc/app/agent/cmd/rpc/agent"
	"github.com/Bluore/mutidoc/pkg/ctxdata"

	"github.com/zeromicro/go-zero/core/logx"
)

// buildCondition 解析前端上下文并补充客户端 IP, 重新序列化后转发给 RPC
func buildCondition(ctx context.Context, raw string) (string, error) {
	cond := agent.ResumeCondition{}
	if raw != "" {
		if err := json.Unmarshal([]byte(raw), &cond); err != nil {
			logx.WithContext(ctx).Errorf("parse condition fail: %v", err)
			return "", httperr.ErrConditionInvalid
		}
	}
	if ip, ok := ctxdata.GetClientIPFromCtx(ctx); ok {
		cond.UserIP = &ip
	}

	b, err := json.Marshal(cond)
	if err != nil {
		logx.WithContext(ctx).Errorf("marshal condition fail: %v", err)
		return "", httperr.ErrConditionInvalid
	}
	return string(b), nil
}
