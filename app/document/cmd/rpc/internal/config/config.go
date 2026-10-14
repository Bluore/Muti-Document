package config

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf
	ServerID int64

	DB struct {
		sqlx.SqlConf
	}
	Cache cache.CacheConf

	// RedisConf 文档未落库变更缓存
	RedisConf RedisConfig
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}
