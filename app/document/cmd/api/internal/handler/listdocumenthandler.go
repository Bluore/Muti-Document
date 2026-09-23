// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package handler

import (
	"net/http"

	"github.com/Bluore/mutidoc/app/document/cmd/api/internal/logic"
	"github.com/Bluore/mutidoc/app/document/cmd/api/internal/svc"
	"github.com/Bluore/mutidoc/app/document/cmd/api/internal/types"
	"github.com/Bluore/mutidoc/pkg/response"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// list document
func listDocumentHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.GetDocumentListReq
		if err := httpx.Parse(r, &req); err != nil {
			response.ResponseErrWithCtx(r.Context(), w, response.ErrValidation)
			return
		}

		l := logic.NewListDocumentLogic(r.Context(), svcCtx)
		resp, err := l.ListDocument(&req)
		response.ResponseWithCtx(r.Context(), w, err, resp)
	}
}
