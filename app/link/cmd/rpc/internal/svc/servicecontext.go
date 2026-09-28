package svc

import (
	"github.com/Bluore/mutidoc/app/link/cmd/rpc/internal/config"
	"github.com/Bluore/mutidoc/app/link/model"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type ServiceContext struct {
	Config config.Config

	ShortLinksModel model.ShortLinksModel
	GrantLinkModel  model.GrantDocumentLinksModel
}

func NewServiceContext(c config.Config) *ServiceContext {
	sqlConn := sqlx.MustNewConn(c.DB.SqlConf)

	return &ServiceContext{
		Config:          c,
		ShortLinksModel: model.NewShortLinksModel(sqlConn, c.Cache),
		GrantLinkModel:  model.NewGrantDocumentLinksModel(sqlConn, c.Cache),
	}
}
