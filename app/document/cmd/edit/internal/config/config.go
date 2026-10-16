// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package config

import (
	"github.com/Bluore/mutidoc/app/document/model"
	"github.com/Bluore/mutidoc/pkg/embedding"
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	rest.RestConf

	// Rpc 编辑服务的 gRPC 配置。ServiceConf 在 main 中复用 RestConf 的配置。
	Rpc zrpc.RpcServerConf

	JwtAuth struct {
		AccessSecret string
		AccessExpire int64
	}

	// RedisConf 协作编辑变更缓存
	RedisConf RedisConfig
	// DB / Cache 文档正文落库使用, 与 document-rpc 保持一致
	DB struct {
		sqlx.SqlConf
	}
	Cache cache.CacheConf

	UserRpcConf zrpc.RpcClientConf

	// Qdrant 文档向量库, 正文落库后重建分块索引使用
	Qdrant model.QdrantConf
	// Embedding 文本向量化(OpenAI 兼容 /embeddings 接口)
	Embedding embedding.Config
	// Chunk 文档切块
	Chunk model.ChunkConf
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}
