package middleware

import (
	"net"
	"net/http"
	"strings"

	"github.com/Bluore/mutidoc/pkg/ctxdata"
)

// ClientIPMiddleware 提取客户端真实 IP 写入上下文
func ClientIPMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if ip := clientIP(r); ip != "" {
			r = r.WithContext(ctxdata.WithClientIP(r.Context(), ip))
		}
		next(w, r)
	}
}

// clientIP 依次从代理头与 RemoteAddr 解析客户端 IP
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if ip := strings.TrimSpace(strings.Split(xff, ",")[0]); ip != "" {
			return ip
		}
	}
	if xri := strings.TrimSpace(r.Header.Get("X-Real-Ip")); xri != "" {
		return xri
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
