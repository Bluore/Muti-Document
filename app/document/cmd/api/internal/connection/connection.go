package connection

import (
	"context"
	"encoding/json"

	"github.com/Bluore/mutidoc/app/document/cmd/api/internal/protocol"
	"github.com/gorilla/websocket"
	"github.com/zeromicro/go-zero/core/logx"
)

type Connection struct {
	UserID     string
	UserDetail protocol.User
	Conn       *websocket.Conn
	Ot         *DocumentClient
	Ctx        context.Context
	UserCxt    context.Context
	Cancel     context.CancelFunc
	Inbox      chan protocol.ClientMessage
	Notify     <-chan struct{}
	Reversion  uint32
}

func NewConnection(
	conn *websocket.Conn,
	userID string,
	userName string,
	ot *DocumentClient,
	userCtx context.Context,
) *Connection {
	ctx, cancel := context.WithCancel(context.Background())
	return &Connection{
		UserID: userID,
		UserDetail: protocol.User{
			ID:   userID,
			Name: userName,
		},
		Conn:    conn,
		Ot:      ot,
		Ctx:     ctx,
		UserCxt: userCtx,
		Cancel:  cancel,
		Inbox:   make(chan protocol.ClientMessage),
	}
}

func (c *Connection) Handle() {
	var sendChannel = make(chan protocol.ServerMessage, 10)
	c.Ot.ConnectUser(c.UserID, c.UserDetail, sendChannel)

	c.sendInitMassage()

	go c.readMessage()

	for {
		c.Notify = c.Ot.GetNotify()

		c.checkAndSendEditHistory()

		select {
		case <-c.Ctx.Done():
			return
		case <-c.UserCxt.Done():
			return
		case <-c.Notify:
			continue
		case message := <-c.Inbox:
			c.handlerMessage(message)
		}
	}

}

func (c *Connection) readMessage() {
	for {
		select {
		case <-c.Ctx.Done():
			return
		case <-c.UserCxt.Done():
			return
		default:
		}

		_, messageRaw, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseAbnormalClosure, websocket.CloseGoingAway) {
				c.Cancel()
				return
			}

			logx.WithContext(c.Ctx).Errorw("error to read msg",
				logx.Field("error", err),
				logx.Field("user_id", c.UserID),
			)
			return
		}

		var message protocol.ClientMessage
		err = json.Unmarshal(messageRaw, &message)
		if err != nil {
			logx.WithContext(c.Ctx).Errorw("error to unmarshal msg",
				logx.Field("error", err),
				logx.Field("user_id", c.UserID),
			)
			continue
		}

		c.Inbox <- message
	}
}

func (c *Connection) handlerMessage(message protocol.ClientMessage) {
	if message.Edit != nil {
		message.Edit.Operator.Reversion = message.Edit.Reversion
		message.Edit.Operator.UserID = c.UserID
		c.Ot.applyEdit(&message.Edit.Operator)
	}
}

func (c *Connection) checkAndSendEditHistory() {
	if c.Reversion >= c.Ot.GetReversion() {
		return
	}

	historyEdit, newReversion := c.Ot.GetHistory(c.Reversion)

	msg := protocol.ServerMessage{
		Edit: &protocol.ServerEditMsg{
			Operator:  historyEdit,
			Reversion: newReversion,
		},
	}

	if err := c.SendMassage(msg); err != nil {
		return
	}

	c.Reversion = newReversion

}

func (c *Connection) SendMassage(msg protocol.ServerMessage) error {
	rawMsg, err := json.Marshal(msg)
	if err != nil {
		logx.WithContext(c.Ctx).Errorw("server edit marshal error",
			logx.Field("error", err),
			logx.Field("edit_msg", msg),
		)
		return err
	}

	logx.WithContext(c.Ctx).Debugw("send msg",
		logx.Field("user_id", c.UserID),
		logx.Field("msg", msg),
	)

	err = c.Conn.WriteMessage(websocket.TextMessage, rawMsg)
	return err
}

func (c *Connection) sendInitMassage() {
	content, reversion := c.Ot.GetContent()
	msg := protocol.ServerMessage{
		Init: &protocol.ServerInitMsg{
			Reversion: reversion,
			Content:   string(content),
			UserID:    c.UserID,
			Users:     c.Ot.GetUsers(),
		},
	}
	c.Reversion = reversion

	c.SendMassage(msg)
}
