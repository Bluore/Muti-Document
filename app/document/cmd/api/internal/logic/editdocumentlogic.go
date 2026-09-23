// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package logic

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Bluore/mutidoc/app/document/cmd/api/internal/connection"
	"github.com/Bluore/mutidoc/app/document/cmd/api/internal/httperr"
	"github.com/Bluore/mutidoc/app/document/cmd/api/internal/protocol"
	"github.com/Bluore/mutidoc/app/document/cmd/api/internal/svc"
	"github.com/Bluore/mutidoc/app/document/cmd/api/internal/types"
	"github.com/gorilla/websocket"
	"github.com/zeromicro/go-zero/core/logx"
)

var (
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
)

type EditDocumentLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// edit document
func NewEditDocumentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *EditDocumentLogic {
	return &EditDocumentLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *EditDocumentLogic) EditDocument(w http.ResponseWriter, r *http.Request, req *types.EditDocumentReq) error {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		l.Errorw("upgrade websocket fail",
			logx.Field("error", err),
			logx.Field("document_id", req.DocumentId),
		)
		return httperr.ErrUpgradeWebSocket
	}

	defer conn.Close()
	l.Infow("wait user", logx.Field("document_id", req.DocumentId))

	var initMessage protocol.ClientInitMessage
	for {
		_, rawMessage, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(
				err,
				websocket.CloseAbnormalClosure,
				websocket.CloseGoingAway,
			) {
				return httperr.ErrUpgradeWebSocket
			}
			l.Errorw("error to read message", logx.Field("error", err))
			return httperr.ErrUpgradeWebSocket
		}

		err = json.Unmarshal(rawMessage, &initMessage)
		if err != nil {
			l.Infow("error to unmarshal init message", logx.Field("error", err))
			continue
		}
		break
	}

	documentClient := l.svcCtx.Document.GetOrCreate(req.DocumentId)

	connHandler := connection.NewConnection(
		conn,
		initMessage.UserID,
		initMessage.UserName,
		documentClient,
		r.Context(),
	)

	l.Infow("join user", logx.Field("user", initMessage))

	connHandler.Handle()
	return nil
}
