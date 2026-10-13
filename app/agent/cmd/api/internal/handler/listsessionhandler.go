// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package handler

import (
	"net/http"

	"github.com/Bluore/mutidoc/app/agent/cmd/api/internal/logic"
	"github.com/Bluore/mutidoc/app/agent/cmd/api/internal/svc"
	"github.com/Bluore/mutidoc/app/agent/cmd/api/internal/types"
	"github.com/Bluore/mutidoc/pkg/response"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// list session
func listSessionHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.ListSessionReq
		if err := httpx.Parse(r, &req); err != nil {
			response.ResponseErrWithCtx(r.Context(), w, response.ErrValidation)
			return
		}

		l := logic.NewListSessionLogic(r.Context(), svcCtx)
		resp, err := l.ListSession(&req)
		response.ResponseWithCtx(r.Context(), w, err, resp)
	}
}
