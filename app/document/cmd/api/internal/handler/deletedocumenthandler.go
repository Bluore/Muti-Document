// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package handler

import (
	"net/http"

	"github.com/Bluore/mutidoc/app/document/cmd/api/internal/logic"
	"github.com/Bluore/mutidoc/app/document/cmd/api/internal/svc"
	"github.com/Bluore/mutidoc/pkg/response"
)

// delete document
func deleteDocumentHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := logic.NewDeleteDocumentLogic(r.Context(), svcCtx)
		err := l.DeleteDocument()
		response.ResponseWithCtx(r.Context(), w, err, nil)
	}
}
