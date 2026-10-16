package ctxdata

import "context"

var (
	CtxKewJwtUserId = "jwtUserID"
	CtxKeyClientIP  = "clientIP"
)

func GetUserIdFromCtx(ctx context.Context) (string, bool) {
	if userID, ok := ctx.Value(CtxKewJwtUserId).(string); ok {
		return userID, true
	}
	return "", false
}

func MustGetUserIdFromCtx(ctx context.Context) string {
	if userID, ok := ctx.Value(CtxKewJwtUserId).(string); ok {
		if userID != "" {
			return userID
		}
	}
	panic("must get user id from ctx fail")
}

// WithClientIP 将客户端 IP 写入上下文
func WithClientIP(ctx context.Context, ip string) context.Context {
	return context.WithValue(ctx, CtxKeyClientIP, ip)
}

// GetClientIPFromCtx 读取上下文中的客户端 IP
func GetClientIPFromCtx(ctx context.Context) (string, bool) {
	ip, ok := ctx.Value(CtxKeyClientIP).(string)
	if !ok || ip == "" {
		return "", false
	}
	return ip, true
}
