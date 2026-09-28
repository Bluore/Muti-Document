// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package handler

import (
	"net/http"

	"github.com/Bluore/mutidoc/app/link/cmd/api/internal/logic"
	"github.com/Bluore/mutidoc/app/link/cmd/api/internal/svc"
	"github.com/Bluore/mutidoc/app/link/cmd/api/internal/types"
	"github.com/Bluore/mutidoc/pkg/globalkey"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// open short link
func GetShortLinkHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.GetShortLinkReq
		if err := httpx.Parse(r, &req); err != nil {
			http.Redirect(w, r, globalkey.UrlNotFound, http.StatusFound)
			return
		}

		l := logic.NewGetShortLinkLogic(r.Context(), svcCtx)
		resp, err := l.GetShortLink(&req)
		if err != nil || resp == nil || resp.Link == "" {
			// 链接不存在/已过期等, 统一跳转到错误页
			http.Redirect(w, r, globalkey.UrlNotFound, http.StatusFound)
			return
		}

		http.Redirect(w, r, resp.Link, http.StatusFound)
	}
}
