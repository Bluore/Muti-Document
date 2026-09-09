// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package info

import (
	"net/http"

	"github.com/Bluore/mutidoc/app/user/cmd/api/internal/logic/info"
	"github.com/Bluore/mutidoc/app/user/cmd/api/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// change user avatar
func AvatarHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := info.NewAvatarLogic(r.Context(), svcCtx)
		err := l.Avatar()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.Ok(w)
		}
	}
}
