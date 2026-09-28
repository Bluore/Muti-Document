package connection

import (
	"sort"
	"sync"

	"github.com/Bluore/mutidoc/app/document/cmd/api/internal/protocol"
	"github.com/Bluore/mutidoc/pkg/ot"
	"github.com/zeromicro/go-zero/core/logx"
)

type DocumentClient struct {
	Users       map[string]protocol.User                 `json:"users"`
	Subscribers map[string]chan<- protocol.ServerMessage `json:"-"`

	Title     string        `json:"title"`
	Content   []byte        `json:"content"`
	Ops       []ot.Operator `json:"ops"`
	Notify    chan struct{} `json:"-"`
	UserMu    sync.RWMutex  `json:"-"`
	ContentMu sync.RWMutex  `json:"-"`
	SendMu    sync.Mutex    `json:"-"`

	Cursors    map[string]ot.Cursor `json:"cursors"`
	CursorLogs [][]ot.Cursor        `json:"cursor_logs"`
	CursorMu   sync.RWMutex         `json:"-"`
}

func NewOt() *DocumentClient {
	return &DocumentClient{
		Users:       make(map[string]protocol.User, 100),
		Subscribers: make(map[string]chan<- protocol.ServerMessage, 100),
		Content:     []byte("helloworld"),
		Ops:         make([]ot.Operator, 0),
		Notify:      make(chan struct{}),
		Cursors:     make(map[string]ot.Cursor, 100),
		CursorLogs:  make([][]ot.Cursor, 0),
	}
}

func (o *DocumentClient) ConnectUser(userID string, userInfo protocol.User, subscriber chan<- protocol.ServerMessage) bool {
	o.UserMu.Lock()

	if _, exist := o.Users[userID]; exist {
		o.UserMu.Unlock()
		return false
	}
	if _, exist := o.Subscribers[userID]; exist {
		o.UserMu.Unlock()
		return false
	}

	o.Users[userID] = userInfo
	o.Subscribers[userID] = subscriber

	o.UserMu.Unlock()

	// Notify the users already in the room that someone joined. The newcomer
	// learns the full roster from its init message, so it is skipped here.
	o.Broadcast(userID, protocol.ServerMessage{
		User: &protocol.ServerUserMsg{
			JoinUsers: []protocol.User{userInfo},
		},
	})

	return true
}

func (o *DocumentClient) DisconnectUser(userID string) {
	o.UserMu.Lock()

	o.CursorMu.Lock()
	userInfo, existed := o.Users[userID]
	delete(o.Users, userID)
	delete(o.Subscribers, userID)
	delete(o.Cursors, userID)
	o.CursorMu.Unlock()

	o.UserMu.Unlock()

	if !existed {
		return
	}

	// Notify the users still in the room that someone left. The user is already
	// gone from Subscribers, so it cannot receive its own quit message.
	o.Broadcast(userID, protocol.ServerMessage{
		User: &protocol.ServerUserMsg{
			QuitUsers: []protocol.User{userInfo},
		},
	})
}

// Broadcast delivers msg to every connected user except exceptUserID (pass an
// empty string to reach everyone). Delivery is best-effort: a full subscriber
// channel is dropped instead of blocking the room.
func (o *DocumentClient) Broadcast(exceptUserID string, msg protocol.ServerMessage) {
	o.UserMu.RLock()
	defer o.UserMu.RUnlock()

	for userID, ch := range o.Subscribers {
		if userID == exceptUserID {
			continue
		}
		select {
		case ch <- msg:
		default:
			logx.Errorw("error to send msg, subscriber channel full",
				logx.Field("user_id", userID),
			)
		}
	}
}

func (o *DocumentClient) NotifyEdit() {
	o.SendMu.Lock()
	defer o.SendMu.Unlock()

	close(o.Notify)
	o.Notify = make(chan struct{})
}

func (o *DocumentClient) GetNotify() <-chan struct{} {
	o.SendMu.Lock()
	defer o.SendMu.Unlock()

	return o.Notify
}

func (o *DocumentClient) GetReversion() uint32 {
	return uint32(len(o.Ops))
}

func (o *DocumentClient) GetHistory(reversion uint32) (ops []ot.Operator, newReversion uint32) {
	o.ContentMu.RLock()
	defer o.ContentMu.RUnlock()

	newReversion = o.GetReversion()
	if reversion >= newReversion {
		return nil, reversion
	}

	return o.Ops[reversion:newReversion], newReversion
}

func (o *DocumentClient) GetContent() (content []byte, newReversion uint32) {
	o.ContentMu.RLock()
	defer o.ContentMu.RUnlock()

	return o.Content, o.GetReversion()
}

func (o *DocumentClient) GetUsers() (users []protocol.User) {
	o.UserMu.RLock()
	defer o.UserMu.RUnlock()

	for _, user := range o.Users {
		users = append(users, user)
	}
	return
}

func (o *DocumentClient) applyEdit(oper *ot.Operator) {
	var err error

	o.ContentMu.Lock()
	defer o.ContentMu.Unlock()

	if oper.Reversion > uint32(len(o.Ops)) {
		logx.Infow("reversion if out of ops",
			logx.Field("reversion", oper.Reversion),
			logx.Field("ops_len", len(o.Ops)),
		)
		return
	}

	var operPrime *ot.Operator
	if oper.Reversion < uint32(len(o.Ops)) {
		operPrime = oper
		for _, historyOper := range o.Ops[oper.Reversion:] {
			logx.Debugw("transform",
				logx.Field("a", operPrime),
				logx.Field("b", historyOper),
			)
			operPrime, _, err = operPrime.Transform(&historyOper)
			if err != nil {
				logx.Errorw("transform error:unsupport operator",
					logx.Field("error", err),
					logx.Field("operator", operPrime),
				)
				return
			}
		}
	} else {
		operPrime = oper
	}

	o.Content = operPrime.Apply(o.Content)
	o.Ops = append(o.Ops, *operPrime)

	o.transformCursors(operPrime)

	o.NotifyEdit()

	logx.Infow("apply edit",
		logx.Field("oper", oper),
		logx.Field("text", string(o.Content)),
	)
}

// transformCursors rebases every stored cursor through oper and records the
// result in CursorLogs. CursorLogs stays aligned with Ops so that the cursor
// changes for a given reversion line up with its operation.
func (o *DocumentClient) transformCursors(oper *ot.Operator) {
	o.CursorMu.Lock()
	defer o.CursorMu.Unlock()

	changed := make([]ot.Cursor, 0)
	for userID, cursor := range o.Cursors {
		oldCursor := cursor
		cursor.ApplyTransform(oper)
		if cursor == oldCursor {
			continue
		}
		o.Cursors[userID] = cursor
		changed = append(changed, cursor)
	}

	if len(changed) == 0 {
		return
	}

	// Keep the log deterministic regardless of map iteration order.
	sort.Slice(changed, func(i, j int) bool {
		return changed[i].UserID < changed[j].UserID
	})

	o.CursorLogs = append(o.CursorLogs, changed)
}

// SetCursor stores the latest cursor reported by a user, rebasing it from the
// reversion the client computed it at to the current one so its offsets stay
// valid after the edits that happened in the meantime. The change is appended
// to CursorLogs and every connection is woken so it gets delivered.
//
// ContentMu is taken before CursorMu on purpose, matching applyEdit's lock
// order: an edit appends an operation and transforms every stored cursor
// atomically, so a cursor must never be stored while an edit is half applied.
func (o *DocumentClient) SetCursor(cursor ot.Cursor) {
	o.ContentMu.RLock()
	o.CursorMu.Lock()

	targetReversion := uint32(len(o.Ops))
	reversion := cursor.Reversion
	if reversion > targetReversion {
		logx.Infow("cursor reversion is ahead of ops",
			logx.Field("user_id", cursor.UserID),
			logx.Field("reversion", cursor.Reversion),
			logx.Field("ops_len", targetReversion),
		)
		reversion = targetReversion
	}

	for i := reversion; i < targetReversion; i++ {
		oper := o.Ops[i]
		cursor.ApplyTransform(&oper)
	}

	cursor.Reversion = targetReversion

	if existing, ok := o.Cursors[cursor.UserID]; ok && existing == cursor {
		o.CursorMu.Unlock()
		o.ContentMu.RUnlock()
		return
	}

	o.Cursors[cursor.UserID] = cursor
	o.CursorLogs = append(o.CursorLogs, []ot.Cursor{cursor})

	o.CursorMu.Unlock()
	o.ContentMu.RUnlock()

	o.NotifyEdit()
}

// GetHistoryCursor returns the cursor changes recorded from version up to the current
// one, mirroring GetHistory for the cursor stream.
func (o *DocumentClient) GetHistoryCursor(version uint32) ([][]ot.Cursor, uint32) {
	o.CursorMu.RLock()
	defer o.CursorMu.RUnlock()

	newVersion := uint32(len(o.CursorLogs))
	if version >= newVersion {
		return make([][]ot.Cursor, 0), version
	}

	return o.CursorLogs[version:newVersion], newVersion
}

func (o *DocumentClient) GetNowCursor() ([]ot.Cursor, uint32) {
	o.CursorMu.RLock()
	defer o.CursorMu.RUnlock()

	cursors := make([]ot.Cursor, 0)
	for _, cursor := range o.Cursors {
		cursors = append(cursors, cursor)
	}
	return cursors, uint32(len(o.CursorLogs))
}

// SetTitle records a new document title and broadcasts the change to every
// connected client (including the one that set it, so all clients converge).
// The title is guarded by ContentMu, which is released before broadcasting:
// Broadcast takes UserMu, and keeping the two lock scopes separate preserves
// the global ContentMu -> UserMu -> CursorMu order used everywhere in this file.
func (o *DocumentClient) SetTitle(userID string, title string) {
	o.ContentMu.Lock()
	if o.Title == title {
		o.ContentMu.Unlock()
		return
	}
	o.Title = title
	o.ContentMu.Unlock()

	o.Broadcast("", protocol.ServerMessage{
		SetTitle: &protocol.ServerSetTitleMsg{
			UserID: userID,
			Title:  title,
		},
	})
}

// GetInitMessage builds the snapshot a newly connected client receives. Content,
// reversion, title, the online users and their cursors are read under one
// overlapping lock scope so the snapshot is internally consistent; reading each
// piece through separate helpers (as the caller used to) could observe an edit
// or a join landing between the reads.
//
// It also returns the cursor log version the snapshot was taken at, so the
// caller can start streaming cursor updates from there instead of replaying the
// whole log (which still contains entries for users that have since left).
//
// UserID and CursorColor are connection-specific, so they are left for the
// caller to fill in.
//
// Locks are taken in ContentMu -> UserMu -> CursorMu order, matching
// DisconnectUser (UserMu -> CursorMu) and applyEdit/SetCursor (ContentMu ->
// CursorMu), so the nesting cannot deadlock.
func (o *DocumentClient) GetInitMessage() (msg protocol.ServerInitMsg, cursorVersion uint32) {
	o.ContentMu.RLock()
	defer o.ContentMu.RUnlock()

	o.UserMu.RLock()
	defer o.UserMu.RUnlock()

	o.CursorMu.RLock()
	defer o.CursorMu.RUnlock()

	users := make([]protocol.User, 0, len(o.Users))
	for _, user := range o.Users {
		users = append(users, user)
	}

	cursors := make([]ot.Cursor, 0, len(o.Cursors))
	for _, cursor := range o.Cursors {
		cursors = append(cursors, cursor)
	}

	return protocol.ServerInitMsg{
		Reversion: uint32(len(o.Ops)),
		Title:     o.Title,
		Content:   string(o.Content),
		Users:     users,
		Cursor:    cursors,
	}, uint32(len(o.CursorLogs))
}
