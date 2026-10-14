package mcp

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/cloudwego/eino-ext/components/tool/mcp/officialmcp"
	"github.com/cloudwego/eino/components/tool"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// 传输类型
const (
	TransportStreamableHTTP = "streamable_http"
)

// MCP 客户端信息
const (
	clientName    = "mutidoc-agent"
	clientVersion = "1.0.0"
)

// 建连超时, 只约束建连阶段
const (
	dialTimeout           = 10 * time.Second
	tlsHandshakeTimeout   = 10 * time.Second
	responseHeaderTimeout = 60 * time.Second
)

// ServerConfig 一个 MCP Server 定义
type ServerConfig struct {
	Name string
	// Type 传输类型, 目前支持 streamable_http
	Type string
	// URL 服务地址
	URL string
	// Headers 形如 "Key: Value", 请求时携带
	Headers []string
}

// LoadTools 连接全部 MCP Server 并返回其工具集
func LoadTools(ctx context.Context, servers []ServerConfig) ([]tool.BaseTool, error) {
	var tools []tool.BaseTool
	for _, s := range servers {
		ts, err := loadServer(ctx, s)
		if err != nil {
			return nil, fmt.Errorf("load mcp server %s: %w", s.Name, err)
		}
		tools = append(tools, ts...)
	}
	return tools, nil
}

func loadServer(ctx context.Context, s ServerConfig) ([]tool.BaseTool, error) {
	switch s.Type {
	case "", TransportStreamableHTTP:
		session, err := connectStreamable(ctx, s)
		if err != nil {
			return nil, err
		}
		// session 需随进程存活, 不主动关闭
		return officialmcp.GetTools(ctx, &officialmcp.Config{Cli: session})
	default:
		return nil, fmt.Errorf("unsupported transport type %q", s.Type)
	}
}

// connectStreamable 建立 streamable http 连接并初始化 MCP 会话
func connectStreamable(ctx context.Context, s ServerConfig) (*sdkmcp.ClientSession, error) {
	if s.URL == "" {
		return nil, fmt.Errorf("mcp url is empty")
	}
	cli := sdkmcp.NewClient(&sdkmcp.Implementation{
		Name:    clientName,
		Version: clientVersion,
	}, nil)

	transport := &sdkmcp.StreamableClientTransport{
		Endpoint:   s.URL,
		HTTPClient: httpClient(s.Headers),
		// 只需请求-响应, 不维持 server->client 的 SSE 长流
		DisableStandaloneSSE: true,
	}
	// 会话与 ctx 生命周期绑定, 这里剥离取消/超时, 否则启动时的超时 ctx 一 cancel 就会断连
	return cli.Connect(context.WithoutCancel(ctx), transport, nil)
}

// httpClient 构造 MCP 请求使用的 http 客户端, 不设置 Client.Timeout 以免长请求被中断
func httpClient(headers []string) *http.Client {
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   dialTimeout,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   tlsHandshakeTimeout,
		ResponseHeaderTimeout: responseHeaderTimeout,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
	}

	var rt http.RoundTripper = transport
	if header := parseHeaders(headers); len(header) > 0 {
		rt = headerRoundTripper{base: transport, headers: header}
	}
	return &http.Client{Transport: rt}
}

func parseHeaders(headers []string) http.Header {
	if len(headers) == 0 {
		return nil
	}
	header := http.Header{}
	for _, h := range headers {
		key, value, ok := strings.Cut(h, ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		header.Set(key, strings.TrimSpace(value))
	}
	return header
}

type headerRoundTripper struct {
	base    http.RoundTripper
	headers http.Header
}

func (h headerRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	for k, vs := range h.headers {
		for _, v := range vs {
			req.Header.Set(k, v)
		}
	}
	return h.base.RoundTrip(req)
}
