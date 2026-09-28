// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package config

import (
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	rest.RestConf
	JwtAuth struct {
		AccessSecret string
		AccessExpire int64
	}
	DocumentRpcConf zrpc.RpcClientConf
	UserRpcConf     zrpc.RpcClientConf
	LinkRpcConf     zrpc.RpcClientConf
}
