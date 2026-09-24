package svc

import (
	"github.com/Bluore/mutidoc/app/document/cmd/rpc/internal/config"
	"github.com/Bluore/mutidoc/app/document/model"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type ServiceContext struct {
	Config config.Config

	DocumentModel model.DocumentsModel
}

func NewServiceContext(c config.Config) *ServiceContext {
	sqlConn := sqlx.MustNewConn(c.DB.SqlConf)

	return &ServiceContext{
		Config:        c,
		DocumentModel: model.NewDocumentsModel(sqlConn, c.Cache),
	}
}
