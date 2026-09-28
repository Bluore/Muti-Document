// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package config

import (
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	rest.RestConf
	BaseUrl string
	JwtAuth struct {
		AccessSecret string
		AccessExpire int64
	}
	LinkRpcConf     zrpc.RpcClientConf
	DocumentRpcConf zrpc.RpcClientConf
}
