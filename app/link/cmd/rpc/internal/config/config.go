package config

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf
	ServerID int64

	// BaseUrl 返回链接时拼接的前缀, 例如 http://localhost:8888
	BaseUrl string

	DB struct {
		sqlx.SqlConf
	}
	Cache cache.CacheConf
}
