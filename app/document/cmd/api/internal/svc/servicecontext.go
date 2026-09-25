// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package svc

import (
	"github.com/Bluore/mutidoc/app/document/cmd/api/internal/config"
	"github.com/Bluore/mutidoc/app/document/cmd/api/internal/types"
	"github.com/Bluore/mutidoc/app/document/cmd/rpc/document"
	"github.com/Bluore/mutidoc/app/user/cmd/rpc/user"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config      config.Config
	Document    *types.DocumentMap
	DocumentRpc document.DocumentZrpcClient
	UserRpc     user.User
}

func NewServiceContext(c config.Config) *ServiceContext {
	documentRpc := document.NewDocumentZrpcClient(zrpc.MustNewClient(c.DocumentRpcConf))
	userRpc := user.NewUser(zrpc.MustNewClient(c.UserRpcConf))

	return &ServiceContext{
		Config:      c,
		Document:    types.NewDocumentMap(),
		DocumentRpc: documentRpc,
		UserRpc:     userRpc,
	}
}
