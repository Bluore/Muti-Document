package connection

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/Bluore/mutidoc/app/document/cmd/edit/internal/protocol"
	"github.com/Bluore/mutidoc/app/document/model"
	"github.com/Bluore/mutidoc/pkg/globalkey"
	"github.com/Bluore/mutidoc/pkg/ot"
	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"
)

const (
	// autoSaveInterval 文档变更写入 redis 的间隔
	autoSaveInterval = 5 * time.Second
	// idleCheckInterval 文档空闲心跳的检查间隔
	idleCheckInterval = time.Second
	// idleTimeout 无用户连接多久后关闭文档协程
	idleTimeout = 10 * time.Second
	// flushEvery 每写入 redis 多少次后落库一次
	flushEvery = 12
	// flushMaxRetry 落库失败后的最大重试次数
	flushMaxRetry = 5
	// flushRetryDelay 落库重试的间隔
	flushRetryDelay = 500 * time.Millisecond
	// storeOpTimeout 单次 redis / 落库操作的超时
	storeOpTimeout = 5 * time.Second
	// flushTimeout 落库(含重试)的整体超时
	flushTimeout = 30 * time.Second
	// closeWait 关闭会话时等待保存协程退出的最长时间
	closeWait = 3 * time.Second
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

	DocumentDetail model.Documents      `json:"-"`
	DocumentModel  model.DocumentsModel `json:"-"`
	RedisClient    *redis.Client        `json:"-"`

	// 以下持久化状态只由 autoDocumentSave 协程读写
	pendingOp      *ot.Operator // 已写入 redis, 尚未落库的净变换
	savedReversion uint32       // pendingOp 已覆盖到的操作版本
	savedTitle     string       // 已持久化的标题
	saveCount      int          // 距离上次落库写入 redis 的次数

	// closing 由 UserMu 保护, 一旦为 true 不再接受新连接
	closing  bool
	stopCh   chan struct{}
	stopOnce sync.Once
	onClose  func(*DocumentClient)
	// onFlushed 在正文成功落库后回调, 用于触发向量索引重建; 可为 nil
	onFlushed func(docID string)

	// ctx 取消后, 进行中的 redis / 落库操作会立即中断; done 在保存协程退出后关闭
	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}
}

// NewOt 创建文档会话。documentDetail 是数据库快照, redisClient / documentModel
// 用于持久化, onClose 在会话空闲关闭后回调(用于从文档表移除), onFlushed 在正文
// 成功落库后回调(用于重建向量索引), 均可为 nil。
// 创建时会先合并 redis 上未落库的变更, 再启动自动保存协程。
func NewOt(
	documentDetail model.Documents,
	redisClient *redis.Client,
	documentModel model.DocumentsModel,
	onClose func(*DocumentClient),
	onFlushed func(docID string),
) *DocumentClient {
	client := &DocumentClient{
		Users:          make(map[string]protocol.User, 100),
		Subscribers:    make(map[string]chan<- protocol.ServerMessage, 100),
		Title:          documentDetail.Summary,
		Content:        []byte(documentDetail.Content),
		Ops:            make([]ot.Operator, 0),
		Notify:         make(chan struct{}),
		Cursors:        make(map[string]ot.Cursor, 100),
		CursorLogs:     make([][]ot.Cursor, 0),
		DocumentDetail: documentDetail,
		DocumentModel:  documentModel,
		RedisClient:    redisClient,
		onClose:        onClose,
		onFlushed:      onFlushed,
		stopCh:         make(chan struct{}),
	}

	client.restoreFromCache()
	client.savedTitle = client.Title

	if client.RedisClient != nil || client.DocumentModel != nil {
		client.ctx, client.cancel = context.WithCancel(context.Background())
		client.done = make(chan struct{})
		go client.autoDocumentSave()
	}
	return client
}

// IsClosed 报告文档会话是否已进入关闭状态
func (o *DocumentClient) IsClosed() bool {
	o.UserMu.RLock()
	defer o.UserMu.RUnlock()
	return o.closing
}

// Close 停止自动保存协程, 用于文档被删除等场景, 不落库。
// 会中断进行中的持久化操作并等待协程退出, 避免删除后被写入覆盖。
func (o *DocumentClient) Close() {
	o.stopOnce.Do(func() {
		o.UserMu.Lock()
		o.closing = true
		o.UserMu.Unlock()
		if o.cancel != nil {
			o.cancel()
		}
		close(o.stopCh)
		// 唤醒所有连接, 让它们感知会话已关闭并结束
		o.NotifyEdit()
	})

	if o.done != nil {
		select {
		case <-o.done:
		case <-time.After(closeWait):
		}
	}
}

func (o *DocumentClient) ConnectUser(userID string, userInfo protocol.User, subscriber chan<- protocol.ServerMessage) bool {
	o.UserMu.Lock()

	if o.closing {
		o.UserMu.Unlock()
		return false
	}
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

// documentCache 是文档在 redis 上的未落库变更。Version 为这批变更落库后的目标版本,
// Ops 为数据库内容到当前内容的净变换(紧凑数组格式)。
type documentCache struct {
	Version int64           `json:"version"`
	Summary string          `json:"summary,omitempty"`
	Ops     json.RawMessage `json:"ops"`
}

// restoreFromCache 在首个用户连接前, 把 redis 上未落库的变更合并进内存内容。
// 只有当 redis 版本大于数据库版本时才认为上次是异常退出。
func (o *DocumentClient) restoreFromCache() {
	if o.RedisClient == nil {
		return
	}

	ctx, cancel := o.opContext(storeOpTimeout)
	defer cancel()

	raw, err := o.RedisClient.Get(ctx, o.changeKey()).Bytes()
	if err == redis.Nil {
		return
	}
	if err != nil {
		logx.Errorw("load document cache fail",
			logx.Field("error", err),
			logx.Field("document_id", o.DocumentDetail.Id),
		)
		return
	}

	var cache documentCache
	if err := json.Unmarshal(raw, &cache); err != nil {
		logx.Errorw("unmarshal document cache fail",
			logx.Field("error", err),
			logx.Field("document_id", o.DocumentDetail.Id),
		)
		return
	}

	// 上次已落库但缓存没删干净, 直接清理
	if cache.Version <= o.DocumentDetail.Version {
		o.deleteCache()
		return
	}

	oper := &ot.Operator{}
	if len(cache.Ops) > 0 {
		decoded, err := decodeOperator(cache.Ops)
		if err != nil {
			logx.Errorw("decode document cache operator fail",
				logx.Field("error", err),
				logx.Field("document_id", o.DocumentDetail.Id),
			)
			return
		}
		oper = decoded
	}

	// 用 redis 上的变更对数据库内容做一次操作变换, 得到初始内容
	if len(oper.Ops) > 0 {
		o.Content = oper.Apply([]byte(o.DocumentDetail.Content))
	}
	if cache.Summary != "" {
		o.Title = cache.Summary
	}
	o.pendingOp = oper

	logx.Infow("restore document from cache",
		logx.Field("document_id", o.DocumentDetail.Id),
		logx.Field("db_version", o.DocumentDetail.Version),
		logx.Field("cache_version", cache.Version),
	)
}

// autoDocumentSave 每 5s 把新增操作合并后写入 redis, 每 12 次落库一次;
// 无用户连接超过 10s 时落库并关闭文档协程。
func (o *DocumentClient) autoDocumentSave() {
	defer close(o.done)

	saveTicker := time.NewTicker(autoSaveInterval)
	defer saveTicker.Stop()

	idleTicker := time.NewTicker(idleCheckInterval)
	defer idleTicker.Stop()

	var idleSince time.Time
	for {
		select {
		case <-o.stopCh:
			return
		case <-saveTicker.C:
			o.persist(false)
		case now := <-idleTicker.C:
			if o.onlineUserCount() > 0 {
				idleSince = time.Time{}
				continue
			}
			if idleSince.IsZero() {
				idleSince = now
				continue
			}
			if now.Sub(idleSince) < idleTimeout {
				continue
			}
			// 关闭前再确认一次没有用户, 避免误关正在使用的文档
			if !o.markClosing() {
				idleSince = time.Time{}
				continue
			}
			o.persist(true)
			if o.onClose != nil {
				o.onClose(o)
			}
			logx.Infow("document session closed by idle timeout",
				logx.Field("document_id", o.DocumentDetail.Id),
			)
			return
		}
	}
}

// persist 把尚未持久化的变更写入 redis; force 为 true 时直接落库(关闭前使用)。
func (o *DocumentClient) persist(force bool) {
	o.ContentMu.RLock()
	current := uint32(len(o.Ops))
	var newOps []ot.Operator
	if current > o.savedReversion {
		newOps = make([]ot.Operator, current-o.savedReversion)
		copy(newOps, o.Ops[o.savedReversion:current])
	}
	content := make([]byte, len(o.Content))
	copy(content, o.Content)
	title := o.Title
	o.ContentMu.RUnlock()

	opsChanged := len(newOps) > 0
	titleChanged := title != o.savedTitle

	if !opsChanged && !titleChanged {
		if force {
			if o.pendingOp != nil {
				if err := o.flushToDatabase(content, title); err != nil {
					logx.Errorw("flush document to db fail",
						logx.Field("error", err),
						logx.Field("document_id", o.DocumentDetail.Id),
					)
					o.saveCache(title)
				} else {
					o.saveCount = 0
				}
			} else {
				o.deleteCache()
			}
		}
		return
	}

	if opsChanged {
		delta, err := composeOperators(newOps)
		if err != nil {
			logx.Errorw("compose document operators fail",
				logx.Field("error", err),
				logx.Field("document_id", o.DocumentDetail.Id),
			)
			return
		}
		if o.pendingOp == nil {
			o.pendingOp = delta
		} else {
			merged, err := o.pendingOp.Compose(delta)
			if err != nil {
				logx.Errorw("merge document operators fail",
					logx.Field("error", err),
					logx.Field("document_id", o.DocumentDetail.Id),
				)
				return
			}
			o.pendingOp = merged
		}
		o.savedReversion = current
	}
	o.savedTitle = title

	// 只有标题变化时也要写入 redis, 用空操作占位
	if o.pendingOp == nil {
		o.pendingOp = &ot.Operator{}
	}

	if !force {
		o.saveCount++
		if o.saveCount%flushEvery != 0 {
			o.saveCache(title)
			return
		}
	}

	if err := o.flushToDatabase(content, title); err != nil {
		logx.Errorw("flush document to db fail",
			logx.Field("error", err),
			logx.Field("document_id", o.DocumentDetail.Id),
		)
		o.saveCache(title)
		return
	}
	o.saveCount = 0
}

// flushToDatabase 把内容与标题落库, 版本冲突时重新读取版本重试, 成功后清理 redis。
func (o *DocumentClient) flushToDatabase(content []byte, title string) error {
	if o.DocumentModel == nil {
		return errors.New("document model not configured")
	}

	ctx, cancel := o.opContext(flushTimeout)
	defer cancel()

	detail := o.DocumentDetail
	detail.Content = string(content)
	detail.Summary = title
	contentChanged := o.DocumentDetail.Content != detail.Content

	version := detail.Version
	var lastErr error
	for attempt := 0; attempt < flushMaxRetry; attempt++ {
		if ctx.Err() != nil {
			break
		}

		detail.Version = version
		err := o.DocumentModel.UpdateWithVersion(ctx, nil, &detail)
		if err == nil {
			o.DocumentDetail = detail
			o.pendingOp = nil
			o.deleteCache()
			logx.Infow("flush document to db success",
				logx.Field("document_id", detail.Id),
				logx.Field("version", detail.Version),
			)
			// 正文变化时重建向量索引; 仅标题变化无需重跑
			if contentChanged {
				o.notifyFlushed(detail.Id)
			}
			return nil
		}
		lastErr = err

		if errors.Is(err, model.ErrNoRowsUpdate) {
			if fresh, findErr := o.DocumentModel.FindOne(ctx, detail.Id); findErr == nil && fresh != nil {
				version = fresh.Version
			}
		}

		// 会话被关闭时立即停止重试, 避免删除后又被写回
		select {
		case <-ctx.Done():
			return lastErr
		case <-time.After(flushRetryDelay):
		}
	}
	return lastErr
}

// saveCache 把当前未落库变更序列化到 redis
func (o *DocumentClient) saveCache(title string) {
	if o.RedisClient == nil || o.pendingOp == nil {
		return
	}

	rawOps, err := encodeOperator(o.pendingOp)
	if err != nil {
		logx.Errorw("encode document operators fail",
			logx.Field("error", err),
			logx.Field("document_id", o.DocumentDetail.Id),
		)
		return
	}

	payload, err := json.Marshal(documentCache{
		Version: o.DocumentDetail.Version + 1,
		Summary: title,
		Ops:     rawOps,
	})
	if err != nil {
		logx.Errorw("marshal document cache fail",
			logx.Field("error", err),
			logx.Field("document_id", o.DocumentDetail.Id),
		)
		return
	}

	ctx, cancel := o.opContext(storeOpTimeout)
	defer cancel()

	if err := o.RedisClient.Set(ctx, o.changeKey(), payload, 0).Err(); err != nil {
		logx.Errorw("save document cache fail",
			logx.Field("error", err),
			logx.Field("document_id", o.DocumentDetail.Id),
		)
	}
}

// deleteCache 落库成功后清理 redis 上的变更
func (o *DocumentClient) deleteCache() {
	if o.RedisClient == nil {
		return
	}

	ctx, cancel := o.opContext(storeOpTimeout)
	defer cancel()

	if err := o.RedisClient.Del(ctx, o.changeKey()).Err(); err != nil {
		logx.Errorw("delete document cache fail",
			logx.Field("error", err),
			logx.Field("document_id", o.DocumentDetail.Id),
		)
	}
}

func (o *DocumentClient) changeKey() string {
	return fmt.Sprintf(globalkey.RedisKeyDocumentChange, o.DocumentDetail.Id)
}

// notifyFlushed 通知正文已落库。回调只负责调度, 需自行异步化并容错。
func (o *DocumentClient) notifyFlushed(docID string) {
	if o.onFlushed == nil {
		return
	}
	o.onFlushed(docID)
}

// opContext 基于会话上下文创建带超时的操作上下文, 会话未启动协程时退化为后台上下文
func (o *DocumentClient) opContext(timeout time.Duration) (context.Context, context.CancelFunc) {
	base := o.ctx
	if base == nil {
		base = context.Background()
	}
	return context.WithTimeout(base, timeout)
}

func (o *DocumentClient) onlineUserCount() int {
	o.UserMu.RLock()
	defer o.UserMu.RUnlock()
	return len(o.Users)
}

// markClosing 在确认没有用户连接后把会话标记为关闭, 防止关闭瞬间又有用户接入
func (o *DocumentClient) markClosing() bool {
	o.UserMu.Lock()
	defer o.UserMu.Unlock()
	if o.closing || len(o.Users) != 0 {
		return false
	}
	o.closing = true
	return true
}

// composeOperators 把有先后顺序的多个操作压缩成一个等价操作
func composeOperators(ops []ot.Operator) (*ot.Operator, error) {
	if len(ops) == 0 {
		return nil, nil
	}

	result := ops[0]
	for i := 1; i < len(ops); i++ {
		next := ops[i]
		composed, err := result.Compose(&next)
		if err != nil {
			return nil, err
		}
		result = *composed
	}
	return &result, nil
}

// encodeOperator 把操作编码成与客户端协议一致的紧凑数组, 便于 redis 反序列化
func encodeOperator(oper *ot.Operator) (json.RawMessage, error) {
	ops := make([]any, 0, len(oper.Ops))
	for _, atomic := range oper.Ops {
		switch v := atomic.(type) {
		case ot.Insert:
			ops = append(ops, v.Str)
		case ot.Retain:
			ops = append(ops, v.N)
		case ot.Delete:
			ops = append(ops, -v.N)
		}
	}
	return json.Marshal(ops)
}

// decodeOperator 复用 ot.Operator.UnmarshalJSON 解析紧凑数组格式
func decodeOperator(raw json.RawMessage) (*ot.Operator, error) {
	oper := &ot.Operator{}
	if err := json.Unmarshal(raw, oper); err != nil {
		return nil, err
	}
	return oper, nil
}
