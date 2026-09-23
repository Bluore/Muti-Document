// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package handler

import (
	"net/http"

	"github.com/Bluore/mutidoc/app/document/cmd/api/internal/logic"
	"github.com/Bluore/mutidoc/app/document/cmd/api/internal/svc"
	"github.com/Bluore/mutidoc/pkg/response"
)

// create document
func createDocumentHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := logic.NewCreateDocumentLogic(r.Context(), svcCtx)
		err := l.CreateDocument()
		response.ResponseWithCtx(r.Context(), w, err, nil)
	}
}
