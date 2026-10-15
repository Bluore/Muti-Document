package logic

import (
	"context"

	"github.com/Bluore/mutidoc/app/document/cmd/edit/internal/connection"
	"github.com/Bluore/mutidoc/app/document/cmd/edit/internal/svc"
	"github.com/Bluore/mutidoc/app/document/cmd/edit/pb"
	"github.com/Bluore/mutidoc/pkg/grpcerr"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetLogic {
	return &GetLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// Get 获取机器人视角的文档最新内容(首次全文, 之后为差异)
func (l *GetLogic) Get(in *pb.GetReq) (*pb.GetResp, error) {
	if in.BotId == "" {
		return nil, grpcerr.REQUEST_PARAM_ERROR
	}

	bot := connection.GetBot(in.BotId)
	if bot == nil {
		return nil, grpcerr.BOT_NOT_EXIST_ERROR
	}

	content, err := bot.GetContent()
	if err != nil {
		logx.WithContext(l.ctx).Errorw("bot get content fail",
			logx.Field("error", err),
			logx.Field("bot_id", in.BotId),
		)
		return nil, grpcerr.SERVER_COMMON_ERROR
	}

	return &pb.GetResp{Content: string(content)}, nil
}
