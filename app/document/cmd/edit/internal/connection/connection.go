package connection

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/Bluore/mutidoc/app/document/cmd/edit/internal/protocol"
	"github.com/Bluore/mutidoc/pkg/ot"
	"github.com/Bluore/mutidoc/pkg/utils"
	"github.com/gorilla/websocket"
	"github.com/zeromicro/go-zero/core/logx"
)

const (
	// writeWait is the time allowed to write a message to the peer.
	writeWait = 10 * time.Second
	// pongWait is how long the server waits for a pong before treating the
	// peer as dead.
	pongWait = 60 * time.Second
	// pingPeriod is how often the server pings the peer. It must be shorter
	// than pongWait so a pong has time to arrive.
	pingPeriod = (pongWait * 9) / 10
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
	Outbox     chan protocol.ServerMessage
	Notify     <-chan struct{}
	Reversion  uint32
	CursorIdx  uint32
	// WriteMu serialises SendMassage: the Handle goroutine and the outbox
	// goroutine both write to the socket and gorilla allows only one
	// concurrent writer (WriteControl is exempt).
	WriteMu sync.Mutex
}

func NewConnection(
	conn *websocket.Conn,
	userID string,
	userName string,
	ot *DocumentClient,
	userCtx context.Context,
) *Connection {
	ctx, cancel := context.WithCancel(context.Background())
	connectionGauge.Inc()
	return &Connection{
		UserID: userID,
		UserDetail: protocol.User{
			ID:          userID,
			Name:        userName,
			CursorColor: utils.RandomCursorColor(),
		},
		Conn:    conn,
		Ot:      ot,
		Ctx:     ctx,
		UserCxt: userCtx,
		Cancel:  cancel,
		Inbox:   make(chan protocol.ClientMessage, 10),
		Outbox:  make(chan protocol.ServerMessage, 10),
	}
}

func (c *Connection) Handle() {
	defer connectionGauge.Dec()

	connSuccess := c.Ot.ConnectUser(c.UserID, c.UserDetail, c.Outbox)
	if !connSuccess {
		logx.WithContext(c.Ctx).Infow("user is already connected, kill conn",
			logx.Field("user_id", c.UserID),
		)
		_ = c.Conn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "user already connected"),
			time.Now().Add(writeWait),
		)
		c.Cancel()
		if err := c.Conn.Close(); err != nil {
			logx.WithContext(c.Ctx).Debugw("error to close conn",
				logx.Field("error", err),
				logx.Field("user_id", c.UserID),
			)
		}
		return
	}
	defer c.killConnection()

	c.sendInitMassage()

	go c.readMessage()
	go c.clientPing()
	go c.listenOutbox()

	for {
		if c.Ot.IsClosed() {
			logx.WithContext(c.Ctx).Infow("document session closed, kill conn",
				logx.Field("user_id", c.UserID),
			)
			return
		}

		c.Notify = c.Ot.GetNotify()

		c.checkAndSendHistory()

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
	_ = c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		return c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	})

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
			c.Cancel()
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
	if message.Cursor != nil {
		message.Cursor.UserID = c.UserID
		c.Ot.SetCursor(*message.Cursor)
	}
	if message.SetTitle != nil {
		c.Ot.SetTitle(c.UserID, message.SetTitle.Title)
	}
}

func (c *Connection) checkAndSendHistory() {
	var (
		msg        protocol.ServerMessage
		hasMessage bool
	)

	historyEdit, newReversion := c.Ot.GetHistory(c.Reversion)

	if len(historyEdit) > 0 {
		msg.Edit = &protocol.ServerEditMsg{
			Operator:  historyEdit,
			Reversion: newReversion,
		}
		c.Reversion = newReversion
		hasMessage = true
	}

	historyCursor, newVersion := c.Ot.GetHistoryCursor(c.CursorIdx)

	if len(historyCursor) > 0 {
		cursorMap := make(map[string]ot.Cursor, len(historyCursor))
		for _, batch := range historyCursor {
			for _, cursor := range batch {
				cursorMap[cursor.UserID] = cursor
			}
		}
		cursors := make([]ot.Cursor, 0, len(cursorMap))
		for _, cursor := range cursorMap {
			cursors = append(cursors, cursor)
		}
		msg.Cursor = &cursors
		c.CursorIdx = newVersion
		hasMessage = true
	}

	if !hasMessage {
		return
	}

	if err := c.SendMassage(msg); err != nil {
		logx.WithContext(c.Ctx).Errorw("error to send history msg, kill conn",
			logx.Field("error", err),
			logx.Field("user_id", c.UserID),
		)
		c.Cancel()
		return
	}
}

func (c *Connection) SendMassage(msg protocol.ServerMessage) error {
	c.WriteMu.Lock()
	defer c.WriteMu.Unlock()

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

	if err := c.Conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
		logx.WithContext(c.Ctx).Errorw("error to set write deadline",
			logx.Field("error", err),
			logx.Field("user_id", c.UserID),
		)
		return err
	}

	err = c.Conn.WriteMessage(websocket.TextMessage, rawMsg)
	return err
}

func (c *Connection) sendInitMassage() {
	initMsg, cursorVersion := c.Ot.GetInitMessage()
	initMsg.UserID = c.UserID
	initMsg.CursorColor = c.UserDetail.CursorColor
	c.Reversion = initMsg.Reversion
	c.CursorIdx = cursorVersion
	msg := protocol.ServerMessage{
		Init: &initMsg,
	}

	if err := c.SendMassage(msg); err != nil {
		logx.WithContext(c.Ctx).Errorw("error to send init msg",
			logx.Field("error", err),
			logx.Field("user_id", c.UserID),
		)
		c.Cancel()
	}
}

func (c *Connection) killConnection() {
	c.Ot.DisconnectUser(c.UserID)
	c.Cancel()
	if err := c.Conn.Close(); err != nil {
		logx.WithContext(c.Ctx).Debugw("error to close conn",
			logx.Field("error", err),
			logx.Field("user_id", c.UserID),
		)
	}
}

// clientPing keeps the connection alive by sending periodic pings. A failed
// ping write means the peer is gone, so the connection is torn down.
func (c *Connection) clientPing() {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-c.Ctx.Done():
			return
		case <-c.UserCxt.Done():
			return
		case <-ticker.C:
			if err := c.Conn.WriteControl(
				websocket.PingMessage,
				nil,
				time.Now().Add(writeWait),
			); err != nil {
				logx.WithContext(c.Ctx).Errorw("error to send ping, kill conn",
					logx.Field("error", err),
					logx.Field("user_id", c.UserID),
				)
				c.Cancel()
				return
			}
		}
	}
}

// listenOutbox consumes the outbox on its own goroutine. It waits for the
// first message, then drains whatever else has queued up behind it, merges the
// batch and sends the result once.
func (c *Connection) listenOutbox() {
	for {
		select {
		case <-c.Ctx.Done():
			return
		case <-c.UserCxt.Done():
			return
		case message := <-c.Outbox:
			c.handleOutbox(message)
		}
	}
}

// handleOutbox drains every message currently waiting in the outbox, folds
// them into a single message and writes it once. It is called from the outbox
// goroutine, so SendMassage serialises the write against the ones made from the
// Handle goroutine.
func (c *Connection) handleOutbox(message protocol.ServerMessage) {
	for {
		select {
		case next := <-c.Outbox:
			message = mergeServerMessage(message, next)
			continue
		default:
		}
		break
	}

	if err := c.SendMassage(message); err != nil {
		logx.WithContext(c.Ctx).Errorw("error to send outbox msg, kill conn",
			logx.Field("error", err),
			logx.Field("user_id", c.UserID),
		)
		c.Cancel()
	}
}

// mergeServerMessage folds two queued broadcast messages into one. User joins
// and quits are de-duplicated by id (later wins, quit beats join) and a title
// update is last-write-wins. Fields already carried by base are preserved, so a
// title update is not dropped when it is coalesced with a user notification.
func mergeServerMessage(base, next protocol.ServerMessage) protocol.ServerMessage {
	newMsg := base
	if next.User != nil {
		joinMap := make(map[string]protocol.User, len(next.User.JoinUsers))
		quitMap := make(map[string]protocol.User, len(next.User.QuitUsers))

		if base.User != nil {
			for _, user := range base.User.JoinUsers {
				joinMap[user.ID] = user
			}
			for _, user := range base.User.QuitUsers {
				quitMap[user.ID] = user
			}
		}

		for _, user := range next.User.JoinUsers {
			joinMap[user.ID] = user
		}
		for _, user := range next.User.QuitUsers {
			quitMap[user.ID] = user
		}

		// A user that quit must not also be reported as joining.
		for id := range quitMap {
			delete(joinMap, id)
		}

		joins := make([]protocol.User, 0, len(joinMap))
		for _, user := range joinMap {
			joins = append(joins, user)
		}
		quits := make([]protocol.User, 0, len(quitMap))
		for _, user := range quitMap {
			quits = append(quits, user)
		}

		newMsg.User = &protocol.ServerUserMsg{
			JoinUsers: joins,
			QuitUsers: quits,
		}
	}

	if next.SetTitle != nil {
		newMsg.SetTitle = next.SetTitle
	}

	return newMsg
}
