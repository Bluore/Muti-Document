// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package svc

import (
	"github.com/Bluore/mutidoc/app/agent/cmd/api/internal/config"
	"github.com/Bluore/mutidoc/app/agent/cmd/rpc/agent"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config   config.Config
	AgentRpc agent.Agent
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config:   c,
		AgentRpc: agent.NewAgent(zrpc.MustNewClient(c.AgentRpcConf)),
	}
}
