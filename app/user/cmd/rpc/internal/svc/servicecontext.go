package svc

import (
	"github.com/Bluore/mutidoc/app/user/cmd/rpc/internal/config"
	"github.com/Bluore/mutidoc/app/user/model"
	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type ServiceContext struct {
	Config config.Config

	RedisClient *redis.Client
	UserModel   model.UsersModel
}

func NewServiceContext(c config.Config) *ServiceContext {
	sqlConn := sqlx.MustNewConn(c.DB.SqlConf)
	redisClient := redis.NewClient(&redis.Options{
		Addr:     c.Redis.Host,
		Password: c.Redis.Pass,
	})

	return &ServiceContext{
		Config:      c,
		RedisClient: redisClient,
		UserModel:   model.NewUsersModel(sqlConn, c.Cache),
	}
}
