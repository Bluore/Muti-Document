package svc

import (
	"github.com/Bluore/mutidoc/app/document/cmd/rpc/internal/config"
	"github.com/Bluore/mutidoc/app/document/model"
	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type ServiceContext struct {
	Config config.Config

	DocumentModel           model.DocumentsModel
	DocumentPermissionModel model.GrantDocumentsPermissionModel

	RedisClient *redis.Client
}

func NewServiceContext(c config.Config) *ServiceContext {
	sqlConn := sqlx.MustNewConn(c.DB.SqlConf)

	redisClient := redis.NewClient(&redis.Options{
		Addr:     c.RedisConf.Addr,
		Password: c.RedisConf.Password,
		DB:       c.RedisConf.DB,
	})

	return &ServiceContext{
		Config:                  c,
		DocumentModel:           model.NewDocumentsModel(sqlConn, c.Cache),
		DocumentPermissionModel: model.NewGrantDocumentsPermissionModel(sqlConn, c.Cache),
		RedisClient:             redisClient,
	}
}
