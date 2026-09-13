package logic

import (
	"context"
	"time"

	"github.com/Bluore/mutidoc/app/user/cmd/rpc/internal/svc"
	"github.com/Bluore/mutidoc/app/user/cmd/rpc/pb"
	"github.com/Bluore/mutidoc/pkg/ctxdata"
	"github.com/Bluore/mutidoc/pkg/grpcerr"
	"github.com/golang-jwt/jwt/v4"
	"github.com/zeromicro/go-zero/core/logx"
)

type GenerateTokenLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGenerateTokenLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GenerateTokenLogic {
	return &GenerateTokenLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 生成Token
func (l *GenerateTokenLogic) GenerateToken(in *pb.GenerateTokenReq) (*pb.GenerateTokenResp, error) {
	now := time.Now()
	expireUnix := now.Add(time.Minute * time.Duration(l.svcCtx.Config.JwtAuth.AccessExpire)).Unix()
	claims := make(jwt.MapClaims)
	claims["iat"] = now.Unix()
	claims["exp"] = expireUnix
	claims[ctxdata.CtxKewJwtUserId] = in.UserId
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(l.svcCtx.Config.JwtAuth.AccessSecret))
	if err != nil {
		logx.WithContext(l.ctx).Errorf("generate token error: %v", err)
		return nil, grpcerr.USER_TOKEN_GERENATE_ERROR
	}

	return &pb.GenerateTokenResp{
		AccessToken:  tokenString,
		AccessExpire: expireUnix,
	}, nil
}
