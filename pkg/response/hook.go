package response

import "net/http"

func AuthHook(w http.ResponseWriter, r *http.Request, err error) {
	ResponseErrWithCtx(r.Context(), w, ErrAuth)
}
