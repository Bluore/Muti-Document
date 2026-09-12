// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package login

import (
	"net/http"

	"github.com/Bluore/mutidoc/app/user/cmd/api/internal/logic/login"
	"github.com/Bluore/mutidoc/app/user/cmd/api/internal/svc"
	"github.com/Bluore/mutidoc/app/user/cmd/api/internal/types"
	"github.com/Bluore/mutidoc/pkg/response"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// get email code
func EmailCodeHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.EmailCodeReq
		if err := httpx.Parse(r, &req); err != nil {
			response.ResponseErrWithCtx(r.Context(), w, response.ErrValidation)
			return
		}

		l := login.NewEmailCodeLogic(r.Context(), svcCtx)
		resp, err := l.EmailCode(&req)
		response.ResponseWithCtx(r.Context(), w, err, resp)
	}
}
