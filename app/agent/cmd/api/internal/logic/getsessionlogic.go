// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package logic

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Bluore/mutidoc/app/agent/cmd/api/internal/httperr"
	"github.com/Bluore/mutidoc/app/agent/cmd/api/internal/svc"
	"github.com/Bluore/mutidoc/app/agent/cmd/api/internal/types"
	"github.com/Bluore/mutidoc/app/agent/cmd/rpc/agent"
	"github.com/Bluore/mutidoc/pkg/globalkey"
	"github.com/Bluore/mutidoc/pkg/grpcerr"
	"github.com/Bluore/mutidoc/pkg/response"

	"github.com/zeromicro/go-zero/core/logx"
)

const (
	// 生成中的轮询间隔
	sessionPollInterval = 50 * time.Millisecond
	// 空闲时的轮询间隔(长连接等待下一轮对话)
	sessionIdlePollInterval = 3 * time.Second
	// 心跳间隔, 保持长连接并探测客户端是否存活
	sessionHeartbeatInterval = 15 * time.Second
)

type GetSessionLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// get session (SSE)
func NewGetSessionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetSessionLogic {
	return &GetSessionLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// GetSession 以 SSE 长连接持续推送会话增量, 跨越多轮对话直到客户端断开或会话被回收
func (l *GetSessionLogic) GetSession(w http.ResponseWriter, r *http.Request, req *types.GetSessionReq) error {
	flusher, ok := w.(http.Flusher)
	if !ok {
		l.Errorw("response writer does not support stream")
		return httperr.ErrStreamUnsupported
	}

	// 支持断点续传: 优先取 query offset, 其次取 SSE Last-Event-ID
	offset := initialOffset(r)

	// 首次拉取放在写响应头之前, 会话不存在等问题可正常返回错误
	first, err := l.svcCtx.AgentRpc.GetSession(l.ctx, &agent.GetSessionReq{
		SessionId: req.SessionID,
		Offset:    offset,
	})
	if err != nil {
		return l.rpcErr(err)
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	// 关闭反向代理缓冲, 保证增量即时下发
	w.Header().Set("X-Accel-Buffering", "no")
	flusher.Flush()

	if err = writeSSE(w, first.GetData(), first.GetNewOffset()); err != nil {
		l.logStreamEnd(req.SessionID, offset, err)
		return nil
	}
	flusher.Flush()
	offset = first.GetNewOffset()
	status := globalkey.SessionStatus(first.GetStatus())

	heartbeat := time.NewTicker(sessionHeartbeatInterval)
	defer heartbeat.Stop()

	// 长连接: 一轮生成结束后不关闭, 空闲等待下一轮 ResumeSession
	for {
		select {
		case <-r.Context().Done():
			l.Infow("session stream client disconnected",
				logx.Field("session_id", req.SessionID),
				logx.Field("offset", offset),
			)
			return nil
		case <-heartbeat.C:
			// 心跳写失败说明客户端已断开, 由心跳决定长连接生命周期
			if err = writeHeartbeat(w); err != nil {
				l.logStreamEnd(req.SessionID, offset, err)
				return nil
			}
			flusher.Flush()
			continue
		case <-time.After(sessionPollDuration(status)):
		}

		result, err := l.svcCtx.AgentRpc.GetSession(l.ctx, &agent.GetSessionReq{
			SessionId: req.SessionID,
			Offset:    offset,
		})
		if err != nil {
			// 响应头已下发, 无法再返回 HTTP 错误; 会话被回收或查询失败时结束长连接
			l.logStreamEnd(req.SessionID, offset, err)
			return nil
		}

		if err = writeSSE(w, result.GetData(), result.GetNewOffset()); err != nil {
			l.logStreamEnd(req.SessionID, offset, err)
			return nil
		}
		flusher.Flush()
		offset = result.GetNewOffset()
		status = globalkey.SessionStatus(result.GetStatus())
	}
}

// rpcErr 把 RPC 错误映射为 HTTP 错误, 仅在响应头下发之前调用
func (l *GetSessionLogic) rpcErr(err error) error {
	if grpcerr.AGENT_SESSION_NOT_EXIST_ERROR.EqualError(err) {
		return httperr.ErrSessionNotExist
	}
	l.Errorf("get session fail: %v", err)
	return response.ErrServer
}

// logStreamEnd 记录长连接结束的原因
func (l *GetSessionLogic) logStreamEnd(sessionID string, offset int64, err error) {
	l.Infow("session stream end",
		logx.Field("session_id", sessionID),
		logx.Field("offset", offset),
		logx.Field("err", err),
	)
}

// initialOffset 解析断点续传游标: 优先 query offset, 其次 SSE Last-Event-ID
func initialOffset(r *http.Request) int64 {
	for _, raw := range []string{r.URL.Query().Get("offset"), r.Header.Get("Last-Event-ID")} {
		if raw == "" {
			continue
		}
		if n, err := strconv.ParseInt(raw, 10, 64); err == nil && n >= 0 {
			return n
		}
	}
	return 0
}

// sessionPollDuration 生成中快轮询, 空闲时慢轮询
func sessionPollDuration(status globalkey.SessionStatus) time.Duration {
	if status == globalkey.SessionStatusGenerating {
		return sessionPollInterval
	}
	return sessionIdlePollInterval
}

// writeSSE 按 SSE 协议写入一条增量事件, 附带断点续传 id
func writeSSE(w io.Writer, data string, id int64) error {
	if data == "" {
		return nil
	}

	var b strings.Builder
	fmt.Fprintf(&b, "id: %d\n", id)
	for _, line := range strings.Split(strings.TrimRight(data, "\n"), "\n") {
		if line == "" {
			continue
		}
		b.WriteString("data: ")
		b.WriteString(line)
		b.WriteByte('\n')
	}
	b.WriteByte('\n')

	_, err := io.WriteString(w, b.String())
	return err
}

// writeHeartbeat 写 SSE 注释行作为心跳, 保持长连接并探测客户端存活
func writeHeartbeat(w io.Writer) error {
	_, err := io.WriteString(w, ": heartbeat\n\n")
	return err
}
