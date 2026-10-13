// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package svc

import (
	"github.com/Bluore/mutidoc/app/document/cmd/api/internal/config"
	"github.com/Bluore/mutidoc/app/document/cmd/edit/edit"
	"github.com/Bluore/mutidoc/app/document/cmd/rpc/document"
	"github.com/Bluore/mutidoc/app/link/cmd/rpc/link"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config      config.Config
	DocumentRpc document.DocumentZrpcClient
	EditRpc     edit.Edit
	LinkRpc     link.Link
}

func NewServiceContext(c config.Config) *ServiceContext {
	documentRpc := document.NewDocumentZrpcClient(zrpc.MustNewClient(c.DocumentRpcConf))
	editRpc := edit.NewEdit(zrpc.MustNewClient(c.EditRpcConf))
	linkRpc := link.NewLink(zrpc.MustNewClient(c.LinkRpcConf))

	return &ServiceContext{
		Config:      c,
		DocumentRpc: documentRpc,
		EditRpc:     editRpc,
		LinkRpc:     linkRpc,
	}
}
