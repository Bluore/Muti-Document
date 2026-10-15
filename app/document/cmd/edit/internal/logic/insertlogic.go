package logic

import (
	"context"
	"errors"

	"github.com/Bluore/mutidoc/app/document/cmd/edit/internal/connection"
	"github.com/Bluore/mutidoc/app/document/cmd/edit/internal/svc"
	"github.com/Bluore/mutidoc/app/document/cmd/edit/pb"
	"github.com/Bluore/mutidoc/pkg/grpcerr"

	"github.com/zeromicro/go-zero/core/logx"
)

type InsertLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewInsertLogic(ctx context.Context, svcCtx *svc.ServiceContext) *InsertLogic {
	return &InsertLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// Insert 在机器人视角文档的 anchor 文本后方插入内容
func (l *InsertLogic) Insert(in *pb.InsertReq) (*pb.InsertResp, error) {
	if in.BotId == "" {
		return nil, grpcerr.REQUEST_PARAM_ERROR
	}

	bot := connection.GetBot(in.BotId)
	if bot == nil {
		return nil, grpcerr.BOT_NOT_EXIST_ERROR
	}

	if err := bot.InsertText(in.Anchor, in.Insertion); err != nil {
		if errors.Is(err, connection.ErrBotTextNotFound) {
			return nil, grpcerr.BOT_TEXT_NOT_FOUND_ERROR
		}
		logx.WithContext(l.ctx).Errorw("bot insert text fail",
			logx.Field("error", err),
			logx.Field("bot_id", in.BotId),
		)
		return nil, grpcerr.SERVER_COMMON_ERROR
	}

	return &pb.InsertResp{}, nil
}
