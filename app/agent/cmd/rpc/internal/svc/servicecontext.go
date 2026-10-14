package svc

import (
	"context"

	agentcore "github.com/Bluore/mutidoc/app/agent/cmd/rpc/agent/agent"
	"github.com/Bluore/mutidoc/app/agent/cmd/rpc/agent/llmmodel"
	"github.com/Bluore/mutidoc/app/agent/cmd/rpc/agent/mcp"
	"github.com/Bluore/mutidoc/app/agent/cmd/rpc/agent/memory"
	"github.com/Bluore/mutidoc/app/agent/cmd/rpc/agent/session"
	agenttool "github.com/Bluore/mutidoc/app/agent/cmd/rpc/agent/tool"
	"github.com/Bluore/mutidoc/app/agent/cmd/rpc/agent/workflow"
	"github.com/Bluore/mutidoc/app/agent/cmd/rpc/internal/config"
	"github.com/Bluore/mutidoc/app/agent/model"
	"github.com/Bluore/mutidoc/app/document/cmd/rpc/document"

	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"github.com/zeromicro/go-zero/zrpc"

	"github.com/cloudwego/eino/components/tool"
)

type ServiceContext struct {
	Config config.Config

	Session session.Config
}

func NewServiceContext(c config.Config) *ServiceContext {
	ctx := context.Background()

	// 加载外部 MCP 工具并挂载到 MainAgent, 失败不阻塞服务启动
	extraTools, err := loadMCPTools(ctx, c.MCP.Servers)
	if err != nil {
		logx.Errorf("load mcp tools fail: %v", err)
	}
	logx.Infof("agent: loaded %d extra tools from %d mcp servers", len(extraTools), len(c.MCP.Servers))

	// 文档工具依赖 document-rpc 校验权限并读取内容
	documentRpc := document.NewDocumentZrpcClient(zrpc.MustNewClient(c.DocumentRpcConf))
	queryDocumentTool, err := agenttool.NewQueryDocumentTool(documentRpc)
	logx.Must(err)
	listDocumentTool, err := agenttool.NewListDocumentTool(documentRpc)
	logx.Must(err)

	// 从配置读取模型密钥, 构建一次顶层智能体供所有会话复用
	ag, err := agentcore.NewAgent(ctx, agentcore.Config{
		Model: llmmodel.DeepSeekConf{
			APIKey:      c.Model.APIKey,
			BaseURL:     c.Model.BaseURL,
			Model:       c.Model.Model,
			Temperature: c.Model.Temperature,
			MaxTokens:   c.Model.MaxTokens,
		},
		DocumentTools: []tool.BaseTool{queryDocumentTool, listDocumentTool},
		ExtraTools:    extraTools,
	})
	logx.Must(err)

	// 记忆缓存: 会话检查点持久化到 redis
	redisClient := redis.NewClient(&redis.Options{
		Addr:     c.Memory.Host,
		Password: c.Memory.Pass,
	})
	memoryStore := memory.NewRedisStore(redisClient,
		memory.WithKeyPrefix(c.Memory.KeyPrefix),
		memory.WithTTL(c.Memory.TTL),
	)

	sqlConn := sqlx.MustNewConn(c.DB.SqlConf)

	return &ServiceContext{
		Config: c,
		Session: session.Config{
			Model:       model.NewAgentSessionsModel(sqlConn, c.Cache),
			Agent:       ag,
			MemoryStore: memoryStore,
			ServerID:    c.ServerID,
			Run:         workflow.Run,
		},
	}
}

// loadMCPTools 按配置连接 MCP Server 并返回其工具
func loadMCPTools(ctx context.Context, servers []config.MCPServerConf) ([]tool.BaseTool, error) {
	if len(servers) == 0 {
		return nil, nil
	}
	confs := make([]mcp.ServerConfig, 0, len(servers))
	for _, s := range servers {
		confs = append(confs, mcp.ServerConfig{
			Name:    s.Name,
			Type:    s.Type,
			URL:     s.URL,
			Headers: s.Headers,
		})
	}

	// 长连接生命周期由 mcp 包内部管理, 不要在此用超时 ctx 取消
	return mcp.LoadTools(ctx, confs)
}
