package connection

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/Bluore/mutidoc/app/document/cmd/edit/internal/protocol"
	"github.com/Bluore/mutidoc/pkg/ot"
	"github.com/zeromicro/go-zero/core/logx"
)

// ErrBotTextNotFound 机器人在其本地快照中找不到待匹配的文本
var ErrBotTextNotFound = errors.New("bot match text not found")

type BotConnection struct {
	UserID    string
	BotID     string
	BotDetail protocol.User
	Ot        *DocumentClient
	Ctx       context.Context
	Cancel    context.CancelFunc
	Outbox    chan protocol.ServerMessage
	Notify    <-chan struct{}
	Reversion uint32
	CursorIdx uint32

	InteractTimes atomic.Int32
	// 最近getOutbox
	BotReversion uint32
	// Content 是机器人视角下的文档内容, 与 BotReversion 对齐,
	// 供 ReplaceText / InsertText 在正确的快照上匹配文本。
	Content []byte

	// Mu 串行化通过 RPC 并发进入的 GetContent / ReplaceText / InsertText
	Mu sync.Mutex
}

func NewBotConnection() *BotConnection {
	ctx, cancel := context.WithCancel(context.Background())
	connectionGauge.Inc()
	return &BotConnection{
		Ctx:    ctx,
		Cancel: cancel,
		Outbox: make(chan protocol.ServerMessage, 10),
	}
}

func (c *BotConnection) Handle() {
	defer connectionGauge.Dec()

	if ok := c.Ot.ConnectUser(c.BotID, c.BotDetail, c.Outbox); !ok {
		logx.WithContext(c.Ctx).Infow("bot is already connected, kill conn",
			logx.Field("bot_id", c.BotID),
		)
		UnregisterBot(c)
		return
	}
	defer c.KillConnection()

	// 机器人没有真实 socket, 只需要维持在线并感知会话关闭
	for {
		if c.Ot.IsClosed() {
			logx.WithContext(c.Ctx).Infow("document session closed, kill bot conn",
				logx.Field("bot_id", c.BotID),
			)
			return
		}

		c.Notify = c.Ot.GetNotify()

		select {
		case <-c.Ctx.Done():
			return
		case <-c.Notify:
		}
	}
}

func (c *BotConnection) KillConnection() {
	UnregisterBot(c)
	c.Ot.DisconnectUser(c.BotID)
	c.Cancel()
}

func (c *BotConnection) getOutbox() (protocol.ServerMessage, bool) {
	var message protocol.ServerMessage
	select {
	case message = <-c.Outbox:
	default:
		return protocol.ServerMessage{}, false
	}

	for {
		select {
		case next := <-c.Outbox:
			message = mergeServerMessage(message, next)
			continue
		default:
		}
		break
	}
	return message, true
}

func (c *BotConnection) GetContent() ([]byte, error) {
	c.Mu.Lock()
	defer c.Mu.Unlock()

	if c.InteractTimes.Load() == 0 {
		c.InteractTimes.Add(1)

		// 消耗冗余消息
		c.getOutbox()
		content, reversion := c.Ot.GetContent()
		c.Content = append([]byte(nil), content...)
		c.Reversion = reversion
		c.BotReversion = reversion

		// 第一次获取，以自然语言返回文档全部信息
		return []byte(fmt.Sprintf("当前文档完整内容如下：\n%s", content)), nil
	}
	c.InteractTimes.Add(1)
	var ops []ot.Operator
	ops, c.Reversion = c.Ot.GetHistory(c.Reversion)
	message, _ := c.getOutbox()
	c.BotReversion = c.Reversion
	// 后面获取，解析msg为自然语言: 文本变更来自 OT 历史, 其余通知来自 outbox
	if len(ops) > 0 {
		message.Edit = &protocol.ServerEditMsg{
			Operator:  ops,
			Reversion: c.Reversion,
		}
		// Init 表示这批变更所基于的旧快照, explainServerMessage 用它还原删除的文本
		message.Init = &protocol.ServerInitMsg{Content: string(c.Content)}

		// 把机器人本地内容推进到最新版本, 与 BotReversion 保持对齐
		for i := range ops {
			c.Content = ops[i].Apply(c.Content)
		}
	}
	return explainServerMessage(message), nil
}

// todo 解析为自然语言
func explainServerMessage(message protocol.ServerMessage) []byte {
	// 如果为空应当提示
	var builder strings.Builder

	// 对于Edit操作：不解析Retain操作，将Delete操作解析为删除的文本，将Insert操作解析为新增的文本
	if message.Edit != nil {
		// 旧快照用于还原删除的文本; 缺失时退化为只报删除长度
		var text []byte
		if message.Init != nil {
			text = []byte(message.Init.Content)
		}

		for _, oper := range message.Edit.Operator {
			next := make([]byte, 0, len(text))
			cursor := 0
			for _, atomic := range oper.Ops {
				switch op := atomic.(type) {
				case ot.Retain:
					if cursor+op.N > len(text) {
						break
					}
					next = append(next, text[cursor:cursor+op.N]...)
					cursor += op.N
				case ot.Insert:
					fmt.Fprintf(&builder, "- 新增文本: %s\n", op.Str)
					next = append(next, op.Str...)
				case ot.Delete:
					if cursor+op.N <= len(text) {
						fmt.Fprintf(&builder, "- 删除文本: %s\n", string(text[cursor:cursor+op.N]))
						cursor += op.N
					} else {
						fmt.Fprintf(&builder, "- 删除文本: %d 个字符\n", op.N)
					}
				}
			}
			text = next
		}
	}

	// 解析其他
	if message.User != nil {
		for _, user := range message.User.JoinUsers {
			name := user.Name
			if name == "" {
				name = user.ID
			}
			fmt.Fprintf(&builder, "- 用户 %s 加入了文档\n", name)
		}
		for _, user := range message.User.QuitUsers {
			name := user.Name
			if name == "" {
				name = user.ID
			}
			fmt.Fprintf(&builder, "- 用户 %s 离开了文档\n", name)
		}
	}

	if message.SetTitle != nil {
		fmt.Fprintf(&builder, "- 文档标题更新为: %s\n", message.SetTitle.Title)
	}

	// Init 仅在独立出现时代表当前全文; 与 Edit 同时出现时它是变更前的快照
	if message.Init != nil && message.Edit == nil {
		fmt.Fprintf(&builder, "- 文档当前内容:\n%s\n", message.Init.Content)
	}

	if builder.Len() == 0 {
		return []byte("文档暂无变化。")
	}
	return []byte(strings.TrimRight(builder.String(), "\n"))
}

// ReplaceText 在机器人本地快照中匹配 original 并替换为 replacement, 提交到 OT。
// 匹配不到时返回 ErrBotTextNotFound。
func (c *BotConnection) ReplaceText(original string, replacement string) error {
	c.Mu.Lock()
	defer c.Mu.Unlock()

	idx := bytes.Index(c.Content, []byte(original))
	if idx < 0 {
		logx.WithContext(c.Ctx).Infow("bot replace text not found",
			logx.Field("bot_id", c.BotID),
			logx.Field("original", original),
		)
		return ErrBotTextNotFound
	}

	oper := ot.Operator{
		UserID:    c.BotID,
		Reversion: c.BotReversion,
		Ops: []ot.OperatorAtomic{
			ot.NewRetain(idx),
			ot.NewDelete(len(original)),
			ot.NewInsert(replacement),
			ot.NewRetain(len(c.Content) - idx - len(original)),
		},
	}
	c.Ot.applyEdit(&oper)
	return nil
}

// InsertText 在机器人本地快照中匹配 anchor 并在其后插入 insertion, 提交到 OT。
// 匹配不到时返回 ErrBotTextNotFound。
func (c *BotConnection) InsertText(anchor string, insertion string) error {
	c.Mu.Lock()
	defer c.Mu.Unlock()

	idx := bytes.Index(c.Content, []byte(anchor))
	if idx < 0 {
		logx.WithContext(c.Ctx).Infow("bot insert anchor not found",
			logx.Field("bot_id", c.BotID),
			logx.Field("anchor", anchor),
		)
		return ErrBotTextNotFound
	}

	pos := idx + len(anchor)
	oper := ot.Operator{
		UserID:    c.BotID,
		Reversion: c.BotReversion,
		Ops: []ot.OperatorAtomic{
			ot.NewRetain(pos),
			ot.NewInsert(insertion),
			ot.NewRetain(len(c.Content) - pos),
		},
	}
	c.Ot.applyEdit(&oper)
	return nil
}
