package logic

import (
	"context"
	"errors"

	"github.com/Bluore/mutidoc/app/document/cmd/edit/internal/connection"
	"github.com/Bluore/mutidoc/app/document/cmd/edit/internal/protocol"
	"github.com/Bluore/mutidoc/app/document/cmd/edit/internal/svc"
	"github.com/Bluore/mutidoc/app/document/cmd/edit/pb"
	"github.com/Bluore/mutidoc/app/document/model"
	"github.com/Bluore/mutidoc/pkg/grpcerr"
	"github.com/Bluore/mutidoc/pkg/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateBotLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateBotLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateBotLogic {
	return &CreateBotLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// CreateBot 创建一个机器人协作连接并加入指定文档
func (l *CreateBotLogic) CreateBot(in *pb.CreateBotReq) (*pb.CreateBotResp, error) {
	if in.DocumentId == "" || in.BotId == "" {
		return nil, grpcerr.REQUEST_PARAM_ERROR
	}

	// 幂等: 同一 BotId 只保留一个连接
	if existing := connection.GetBot(in.BotId); existing != nil {
		return &pb.CreateBotResp{}, nil
	}

	documentClient := l.svcCtx.Document.Get(in.DocumentId)
	if documentClient == nil {
		documentDetail, err := l.svcCtx.DocumentModel.FindOne(l.ctx, in.DocumentId)
		if err != nil {
			if errors.Is(err, model.ErrNotFound) {
				return nil, grpcerr.DOCUMENT_NOT_EXIST_ERROR
			}
			logx.WithContext(l.ctx).Errorw("query document fail",
				logx.Field("error", err),
				logx.Field("document_id", in.DocumentId),
			)
			return nil, grpcerr.DB_ERROR
		}
		documentClient = l.svcCtx.Document.GetOrCreate(in.DocumentId, *documentDetail)
	}

	botName := in.BotName
	if botName == "" {
		botName = in.BotId
	}

	bot := connection.NewBotConnection()
	bot.UserID = in.UserId
	bot.BotID = in.BotId
	bot.BotDetail = protocol.User{
		ID:          in.BotId,
		Name:        botName,
		CursorColor: utils.RandomCursorColor(),
	}
	bot.Ot = documentClient

	connection.RegisterBot(bot)
	go bot.Handle()

	logx.WithContext(l.ctx).Infow("create bot connection",
		logx.Field("bot_id", in.BotId),
		logx.Field("document_id", in.DocumentId),
	)

	return &pb.CreateBotResp{}, nil
}
