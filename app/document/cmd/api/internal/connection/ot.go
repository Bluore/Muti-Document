package connection

import (
	"sync"

	"github.com/Bluore/mutidoc/app/document/cmd/api/internal/protocol"
	"github.com/Bluore/mutidoc/pkg/ot"
	"github.com/zeromicro/go-zero/core/logx"
)

type DocumentClient struct {
	Users       map[string]protocol.User                 `json:"users"`
	Subscribers map[string]chan<- protocol.ServerMessage `json:"-"`

	Content   []byte        `json:"content"`
	Ops       []ot.Operator `json:"ops"`
	Notify    chan struct{} `json:"-"`
	UserMu    sync.RWMutex  `json:"-"`
	ContentMu sync.RWMutex  `json:"-"`
	SendMu    sync.Mutex    `json:"-"`
}

func NewOt() *DocumentClient {
	return &DocumentClient{
		Users:       make(map[string]protocol.User, 100),
		Subscribers: make(map[string]chan<- protocol.ServerMessage, 100),
		Content:     []byte("helloworld"),
		Ops:         make([]ot.Operator, 0),
		Notify:      make(chan struct{}),
	}
}

func (o *DocumentClient) ConnectUser(userID string, userInfo protocol.User, subscriber chan<- protocol.ServerMessage) {

	o.UserMu.Lock()
	defer o.UserMu.Unlock()

	o.Users[userID] = userInfo
	o.Subscribers[userID] = subscriber
}

func (o *DocumentClient) Broadcase(msg protocol.ServerMessage) {
	for userID, ch := range o.Subscribers {
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

	o.NotifyEdit()

	logx.Infow("apply edit",
		logx.Field("oper", oper),
		logx.Field("text", string(o.Content)),
	)
}
