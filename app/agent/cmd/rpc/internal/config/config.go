package config

import (
	"time"

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

	Model  ModelConf
	Memory MemoryConf
	MCP    MCPConf `json:",optional"`

	// DocumentRpcConf document-rpc 客户端配置, 供文档工具查询内容与校验权限
	DocumentRpcConf zrpc.RpcClientConf
	// EditRpcConf edit-rpc 客户端配置, 供文档编辑工具建立机器人连接并提交 OT 编辑
	EditRpcConf zrpc.RpcClientConf
}

// ModelConf 对话模型配置
type ModelConf struct {
	APIKey      string
	BaseURL     string
	Model       string
	Temperature float32
	MaxTokens   int
}

// MCPConf 外部 MCP Server 配置
type MCPConf struct {
	Servers []MCPServerConf `json:",optional"`
}

// MCPServerConf 单个 MCP Server
type MCPServerConf struct {
	Name string
	// Type 传输类型, 目前支持 streamable_http, 空值按 streamable_http 处理
	Type string `json:",optional"`
	// URL 服务地址
	URL string
	// Headers 形如 "Key: Value"
	Headers []string `json:",optional"`
}

// MemoryConf 记忆缓存 (Redis) 配置
type MemoryConf struct {
	Host      string
	Pass      string
	KeyPrefix string
	TTL       time.Duration
}
