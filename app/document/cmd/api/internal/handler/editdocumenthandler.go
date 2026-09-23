// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package handler

import (
	"net/http"

	"github.com/Bluore/mutidoc/app/document/cmd/api/internal/logic"
	"github.com/Bluore/mutidoc/app/document/cmd/api/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// edit document
func editDocumentHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := logic.NewEditDocumentLogic(r.Context(), svcCtx)
		err := l.EditDocument()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.Ok(w)
		}
	}
}
