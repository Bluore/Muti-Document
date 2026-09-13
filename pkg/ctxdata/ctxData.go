package ctxdata

import "context"

var CtxKewJwtUserId = "jwtUserID"

func GetUserIdFromCtx(ctx context.Context) (string, bool) {
	if userID, ok := ctx.Value(CtxKewJwtUserId).(string); ok {
		return userID, true
	}
	return "", false
}
