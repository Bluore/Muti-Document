package svc

import (
	"github.com/Bluore/mutidoc/app/document/cmd/rpc/internal/config"
	"github.com/Bluore/mutidoc/app/document/model"
	"github.com/Bluore/mutidoc/pkg/embedding"
	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type ServiceContext struct {
	Config config.Config

	DocumentModel           model.DocumentsModel
	DocumentPermissionModel model.GrantDocumentsPermissionModel

	// DocumentChunks 文档分块向量表, 用于相似内容检索; 向量库不可用时为 nil
	DocumentChunks model.DocumentChunksModel
	// Embedder 把检索文本向量化
	Embedder model.Embedder

	RedisClient *redis.Client
}

func NewServiceContext(c config.Config) *ServiceContext {
	sqlConn := sqlx.MustNewConn(c.DB.SqlConf)

	redisClient := redis.NewClient(&redis.Options{
		Addr:     c.RedisConf.Addr,
		Password: c.RedisConf.Password,
		DB:       c.RedisConf.DB,
	})

	svcCtx := &ServiceContext{
		Config:                  c,
		DocumentModel:           model.NewDocumentsModel(sqlConn, c.Cache),
		DocumentPermissionModel: model.NewGrantDocumentsPermissionModel(sqlConn, c.Cache),
		Embedder:                embedding.NewClient(c.Embedding),
		RedisClient:             redisClient,
	}

	chunksModel, err := model.NewDocumentChunksModel(c.Qdrant)
	if err != nil {
		logx.Errorf("init document chunks model fail, vector search disabled: %v", err)
	} else {
		svcCtx.DocumentChunks = chunksModel
	}
	return svcCtx
}
