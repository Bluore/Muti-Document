// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package svc

import (
	"github.com/Bluore/mutidoc/app/document/cmd/api/internal/config"
	"github.com/Bluore/mutidoc/app/document/cmd/api/internal/types"
	"github.com/Bluore/mutidoc/app/document/cmd/rpc/document"
	"github.com/Bluore/mutidoc/app/document/model"
	"github.com/Bluore/mutidoc/app/link/cmd/rpc/link"
	"github.com/Bluore/mutidoc/app/user/cmd/rpc/user"
	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config      config.Config
	Document    *types.DocumentMap
	DocumentRpc document.DocumentZrpcClient
	UserRpc     user.User
	LinkRpc     link.Link

	RedisClient   *redis.Client
	DocumentModel model.DocumentsModel
}

func NewServiceContext(c config.Config) *ServiceContext {
	documentRpc := document.NewDocumentZrpcClient(zrpc.MustNewClient(c.DocumentRpcConf))
	userRpc := user.NewUser(zrpc.MustNewClient(c.UserRpcConf))
	linkRpc := link.NewLink(zrpc.MustNewClient(c.LinkRpcConf))

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
		DocumentRpc:   documentRpc,
		UserRpc:       userRpc,
		LinkRpc:       linkRpc,
		RedisClient:   redisClient,
		DocumentModel: documentModel,
	}
}
