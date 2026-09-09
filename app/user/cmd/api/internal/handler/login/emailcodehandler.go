// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package login

import (
	"net/http"

	"github.com/Bluore/mutidoc/app/user/cmd/api/internal/logic/login"
	"github.com/Bluore/mutidoc/app/user/cmd/api/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// get email code
func EmailCodeHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := login.NewEmailCodeLogic(r.Context(), svcCtx)
		err := l.EmailCode()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.Ok(w)
		}
	}
}
