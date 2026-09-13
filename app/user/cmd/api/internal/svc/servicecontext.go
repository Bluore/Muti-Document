// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package svc

import (
	"github.com/Bluore/mutidoc/app/user/cmd/api/internal/config"
	"github.com/Bluore/mutidoc/app/user/cmd/rpc/user"
	"github.com/Bluore/mutidoc/pkg/emailcontrol"
	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config      config.Config
	EmailSender emailcontrol.EmailSender
	RedisClient *redis.Client
	UserRpc     user.User
}

func NewServiceContext(c config.Config) *ServiceContext {
	if c.EmailSender.Platform != "resend" {
		panic("email platform not supported")
	}
	emailClient := emailcontrol.NewResendClient(c.EmailSender.ApiKey)

	redisClient := redis.NewClient(&redis.Options{
		Addr:     c.RedisConf.Addr,
		Password: c.RedisConf.Password,
		DB:       c.RedisConf.DB,
	})

	userRpc := user.NewUser(zrpc.MustNewClient(c.UserRpcConf))

	return &ServiceContext{
		Config:      c,
		EmailSender: emailClient,
		RedisClient: redisClient,
		UserRpc:     userRpc,
	}
}
