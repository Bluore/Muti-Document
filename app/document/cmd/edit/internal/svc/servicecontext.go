// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package svc

import (
	"github.com/Bluore/mutidoc/app/document/cmd/edit/internal/config"
	"github.com/Bluore/mutidoc/app/document/cmd/edit/internal/types"
	"github.com/Bluore/mutidoc/app/document/model"
	"github.com/Bluore/mutidoc/app/user/cmd/rpc/user"
	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config   config.Config
	Document *types.DocumentMap
	UserRpc  user.User

	RedisClient   *redis.Client
	DocumentModel model.DocumentsModel
}

func NewServiceContext(c config.Config) *ServiceContext {
	userRpc := user.NewUser(zrpc.MustNewClient(c.UserRpcConf))

	redisClient := redis.NewClient(&redis.Options{
		Addr:     c.RedisConf.Addr,
		Password: c.RedisConf.Password,
		DB:       c.RedisConf.DB,
	})

	sqlConn := sqlx.MustNewConn(c.DB.SqlConf)
	documentModel := model.NewDocumentsModel(sqlConn, c.Cache)

	return &ServiceContext{
		Config:        c,
		Document:      types.NewDocumentMap(redisClient, documentModel),
		UserRpc:       userRpc,
		RedisClient:   redisClient,
		DocumentModel: documentModel,
	}
}
