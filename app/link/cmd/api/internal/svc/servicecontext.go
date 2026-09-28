// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package svc

import (
	"github.com/Bluore/mutidoc/app/link/cmd/api/internal/config"
	"github.com/Bluore/mutidoc/app/link/cmd/rpc/link"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config  config.Config
	LinkRpc link.Link
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config:  c,
		LinkRpc: link.NewLink(zrpc.MustNewClient(c.LinkRpcConf)),
	}
}
