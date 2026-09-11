package rpcserver

import (
	"context"
	"time"

	"github.com/Bluore/mutidoc/pkg/grpcerr"
	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func LoggingInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	start := time.Now()
	resp, err := handler(ctx, req)

	logx.WithContext(ctx).Infow("rpc call",
		logx.Field("method", info.FullMethod),
		logx.Field("duration", time.Since(start).String()),
		logx.Field("error", err),
	)
	if err == nil {
		return resp, nil
	}
	var grpcErr grpcerr.ErrCode
	if !errors.As(err, &grpcErr) {
		logx.WithContext(ctx).Errorw("rpc call, error unknow")
		return resp, status.Error(codes.Code(grpcerr.SERVER_COMMON_ERROR), "rpc call failed")
	}
	return resp, status.Error(codes.Code(grpcErr.Code()), grpcErr.Msg())
}
