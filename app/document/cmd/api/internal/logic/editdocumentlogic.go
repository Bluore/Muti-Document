// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package logic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/Bluore/mutidoc/app/document/cmd/api/internal/connection"
	"github.com/Bluore/mutidoc/app/document/cmd/api/internal/httperr"
	"github.com/Bluore/mutidoc/app/document/cmd/api/internal/protocol"
	"github.com/Bluore/mutidoc/app/document/cmd/api/internal/svc"
	"github.com/Bluore/mutidoc/app/document/cmd/api/internal/types"
	"github.com/Bluore/mutidoc/app/document/model"
	"github.com/Bluore/mutidoc/app/user/cmd/rpc/user"
	"github.com/Bluore/mutidoc/pkg/ctxdata"
	"github.com/Bluore/mutidoc/pkg/grpcerr"
	"github.com/Bluore/mutidoc/pkg/response"
	"github.com/golang-jwt/jwt/v4"
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

	// check jwt
	ctx, err := l.parseToken(initMessage.Token)
	if err != nil {
		l.Errorw("check jwt fail",
			logx.Field("error", err),
			logx.Field("document_id", req.DocumentId),
		)
		return response.ErrAuth
	}

	// query userInfo
	userID := ctxdata.MustGetUserIdFromCtx(ctx)
	userDetail, err := l.svcCtx.UserRpc.GetUserDetail(ctx, &user.GetUserDetailReq{
		UserID: userID,
	})
	if err != nil {
		if grpcerr.USER_NOT_EXIST_ERROR.EqualError(err) {
			return httperr.ErrUserNotExist
		}
		l.Errorw("query user info fail",
			logx.Field("error", err),
			logx.Field("user_id", userID),
		)
		return response.ErrServer
	}

	// todo but not now check grant

	// 读取数据库快照, 交给文档会话与 redis 上未落库的变更合并
	documentDetail, err := l.svcCtx.DocumentModel.FindOne(ctx, req.DocumentId)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return httperr.ErrDocumentNotExist
		}
		l.Errorw("query document fail",
			logx.Field("error", err),
			logx.Field("document_id", req.DocumentId),
		)
		return response.ErrServer
	}

	documentClient := l.svcCtx.Document.GetOrCreate(req.DocumentId, *documentDetail)

	connHandler := connection.NewConnection(
		conn,
		userID,
		userDetail.NickName,
		documentClient,
		r.Context(),
	)

	l.Infow("join user", logx.Field("user", connHandler.UserDetail))

	connHandler.Handle()
	return nil
}

// parseToken validates the jwt token, stores the user id carried in its claims
// into a new context and returns it. The user id can then be read with
// ctxdata.GetUserIdFromCtx / ctxdata.MustGetUserIdFromCtx.
func (l *EditDocumentLogic) parseToken(tokenString string) (context.Context, error) {
	if tokenString == "" {
		return nil, errors.New("empty token")
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(l.svcCtx.Config.JwtAuth.AccessSecret), nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}

	userID, ok := claims[ctxdata.CtxKewJwtUserId].(string)
	if !ok || userID == "" {
		return nil, errors.New("token missing user id")
	}

	return context.WithValue(l.ctx, ctxdata.CtxKewJwtUserId, userID), nil
}
