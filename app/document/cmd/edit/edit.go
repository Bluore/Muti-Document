// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package main

import (
	"flag"

	"github.com/Bluore/mutidoc/app/document/cmd/edit/internal/config"
	"github.com/Bluore/mutidoc/app/document/cmd/edit/internal/handler"
	"github.com/Bluore/mutidoc/app/document/cmd/edit/internal/server"
	"github.com/Bluore/mutidoc/app/document/cmd/edit/internal/svc"
	"github.com/Bluore/mutidoc/app/document/cmd/edit/pb"
	"github.com/Bluore/mutidoc/pkg/interceptor/rpcserver"
	"github.com/Bluore/mutidoc/pkg/response"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/edit.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)
	// REST 与 RPC 同进程, 复用同一份服务配置(日志/模式/监控/链路), 避免二次初始化覆盖
	c.Rpc.ServiceConf = c.RestConf.ServiceConf

	ctx := svc.NewServiceContext(c)

	restServer := rest.MustNewServer(
		c.RestConf,
		rest.WithUnauthorizedCallback(response.AuthHook),
	)
	handler.RegisterHandlers(restServer, ctx)

	rpcServer := zrpc.MustNewServer(c.Rpc, func(grpcServer *grpc.Server) {
		pb.RegisterEditServer(grpcServer, server.NewEditServer(ctx))

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	rpcServer.AddUnaryInterceptors(rpcserver.LoggingInterceptor)

	group := service.NewServiceGroup()
	defer group.Stop()
	group.Add(restServer)
	group.Add(rpcServer)

	logx.Infof("Starting edit-api: rest %s:%d, rpc %s", c.Host, c.Port, c.Rpc.ListenOn)
	group.Start()
}
