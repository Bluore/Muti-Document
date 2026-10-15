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

type ReplaceLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewReplaceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ReplaceLogic {
	return &ReplaceLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// Replace 替换机器人视角文档中的文本
func (l *ReplaceLogic) Replace(in *pb.ReplaceReq) (*pb.ReplaceResp, error) {
	if in.BotId == "" {
		return nil, grpcerr.REQUEST_PARAM_ERROR
	}

	bot := connection.GetBot(in.BotId)
	if bot == nil {
		return nil, grpcerr.BOT_NOT_EXIST_ERROR
	}

	if err := bot.ReplaceText(in.Original, in.Replacement); err != nil {
		if errors.Is(err, connection.ErrBotTextNotFound) {
			return nil, grpcerr.BOT_TEXT_NOT_FOUND_ERROR
		}
		logx.WithContext(l.ctx).Errorw("bot replace text fail",
			logx.Field("error", err),
			logx.Field("bot_id", in.BotId),
		)
		return nil, grpcerr.SERVER_COMMON_ERROR
	}

	return &pb.ReplaceResp{}, nil
}
