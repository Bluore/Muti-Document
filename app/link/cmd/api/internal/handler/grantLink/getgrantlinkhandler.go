// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package grantLink

import (
	"net/http"

	"github.com/Bluore/mutidoc/app/link/cmd/api/internal/logic/grantLink"
	"github.com/Bluore/mutidoc/app/link/cmd/api/internal/svc"
	"github.com/Bluore/mutidoc/app/link/cmd/api/internal/types"
	"github.com/Bluore/mutidoc/pkg/response"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// grant document permission and open link
func GetGrantLinkHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.GetGrantDocumentReq
		if err := httpx.Parse(r, &req); err != nil {
			response.ResponseErrWithCtx(r.Context(), w, response.ErrValidation)
			return
		}

		l := grantLink.NewGetGrantLinkLogic(r.Context(), svcCtx)
		resp, err := l.GetGrantLink(&req)
		response.ResponseWithCtx(r.Context(), w, err, resp)
	}
}
