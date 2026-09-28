// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package grantLink

import (
	"net/http"

	"github.com/Bluore/mutidoc/app/link/cmd/api/internal/logic/grantLink"
	"github.com/Bluore/mutidoc/app/link/cmd/api/internal/svc"
	"github.com/Bluore/mutidoc/app/link/cmd/api/internal/types"
	"github.com/Bluore/mutidoc/pkg/globalkey"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// grant document permission and open link
func GetGrantLinkHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.GetGrantDocumentReq
		if err := httpx.Parse(r, &req); err != nil {
			http.Redirect(w, r, globalkey.UrlLogin, http.StatusFound)
			return
		}

		l := grantLink.NewGetGrantLinkLogic(r.Context(), svcCtx)
		resp, err := l.GetGrantLink(&req)
		if err != nil || resp == nil || resp.Link == "" {
			// 链接不存在/已过期/无权限等, 统一跳转到错误页
			http.Redirect(w, r, globalkey.UrlNotFound, http.StatusFound)
			return
		}

		http.Redirect(w, r, resp.Link, http.StatusFound)
	}
}
