package logic

import (
	"context"

	"github.com/Bluore/mutidoc/app/document/cmd/edit/internal/connection"
	"github.com/Bluore/mutidoc/app/document/cmd/edit/internal/svc"
	"github.com/Bluore/mutidoc/app/document/cmd/edit/pb"
	"github.com/Bluore/mutidoc/pkg/grpcerr"

	"github.com/zeromicro/go-zero/core/logx"
)

type QuitBotConnectionLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewQuitBotConnectionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QuitBotConnectionLogic {
	return &QuitBotConnectionLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// QuitBotConnection 机器人主动退出文档协作连接, 幂等: 连接已不存在时视为已退出
func (l *QuitBotConnectionLogic) QuitBotConnection(in *pb.QuitBotConnectionReq) (*pb.QuitBotConnectionResp, error) {
	if in.BotId == "" {
		return nil, grpcerr.REQUEST_PARAM_ERROR
	}

	if bot := connection.GetBot(in.BotId); bot != nil {
		bot.KillConnection()
		logx.WithContext(l.ctx).Infow("quit bot connection",
			logx.Field("bot_id", in.BotId),
		)
	}

	return &pb.QuitBotConnectionResp{}, nil
}
