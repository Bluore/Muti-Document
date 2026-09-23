// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package svc

import (
	"github.com/Bluore/mutidoc/app/document/cmd/api/internal/config"
	"github.com/Bluore/mutidoc/app/document/cmd/api/internal/types"
)

type ServiceContext struct {
	Config   config.Config
	Document *types.DocumentMap
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config:   c,
		Document: types.NewDocumentMap(),
	}
}
