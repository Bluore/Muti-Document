package response

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"
)

type RespErr struct {
	HttpCode int    `json:"-"`
	Code     int    `json:"code"`
	Msg      string `json:"msg"`
	Data     any    `json:"data"`
}

type response struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data any    `json:"data"`
}

func (r RespErr) Error() string {
	return fmt.Sprintf("response Error: code:%d msg:%s", r.Code, r.Msg)
}

func NewError(status int, code int, msg string) error {
	return RespErr{
		HttpCode: status,
		Code:     code,
		Msg:      msg,
		Data:     make(map[string]any),
	}
}

func ResponseErrWithCtx(ctx context.Context, w http.ResponseWriter, err error) {
	var resp RespErr
	if ok := errors.As(err, &resp); !ok {
		logx.WithContext(ctx).
			Errorf("response error not support: %+v", err)
		httpx.WriteJsonCtx(ctx, w, http.StatusInternalServerError, map[string]any{})
		return
	}
	logx.WithContext(ctx).
		Infof("response error: %+v", resp)

	httpx.WriteJsonCtx(ctx, w, resp.HttpCode, resp)
}

func ResponseWithCtx(ctx context.Context, w http.ResponseWriter, err error, data any) {
	if err != nil {
		ResponseErrWithCtx(ctx, w, err)
		return
	}

	httpx.WriteJsonCtx(ctx, w, http.StatusOK, response{
		Code: 0,
		Msg:  "",
		Data: data,
	})
}
