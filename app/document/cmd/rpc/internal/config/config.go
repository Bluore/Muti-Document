package config

import (
	"github.com/Bluore/mutidoc/app/document/model"
	"github.com/Bluore/mutidoc/pkg/embedding"
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

	// Qdrant 文档向量库, 相似内容检索使用
	Qdrant model.QdrantConf
	// Embedding 文本向量化(OpenAI 兼容 /embeddings 接口)
	Embedding embedding.Config
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}
