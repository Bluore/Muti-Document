// Package botconn 追踪会话中已建立的机器人编辑连接, 供 EditAgent 结束后统一退出。
package botconn

import (
	"context"
	"sync"
)

type trackerKey struct{}

// Tracker 记录会话中尚未退出的机器人连接
type Tracker struct {
	mu  sync.Mutex
	ids map[string]struct{}
}

// WithTracker 在 ctx 上挂载连接追踪器, 已存在时复用
func WithTracker(ctx context.Context) context.Context {
	if _, ok := ctx.Value(trackerKey{}).(*Tracker); ok {
		return ctx
	}
	return context.WithValue(ctx, trackerKey{}, &Tracker{ids: make(map[string]struct{})})
}

// Track 记录一个已建立的机器人连接; ctx 上没有追踪器时忽略
func Track(ctx context.Context, botID string) {
	if botID == "" {
		return
	}
	t, ok := ctx.Value(trackerKey{}).(*Tracker)
	if !ok {
		return
	}
	t.mu.Lock()
	t.ids[botID] = struct{}{}
	t.mu.Unlock()
}

// Drain 取出并清空已记录的机器人连接
func Drain(ctx context.Context) []string {
	t, ok := ctx.Value(trackerKey{}).(*Tracker)
	if !ok {
		return nil
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	ids := make([]string, 0, len(t.ids))
	for id := range t.ids {
		ids = append(ids, id)
	}
	t.ids = make(map[string]struct{})
	return ids
}
