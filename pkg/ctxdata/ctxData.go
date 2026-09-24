package ctxdata

import "context"

var CtxKewJwtUserId = "jwtUserID"

func GetUserIdFromCtx(ctx context.Context) (string, bool) {
	if userID, ok := ctx.Value(CtxKewJwtUserId).(string); ok {
		return userID, true
	}
	return "", false
}

func MustGetUserIdFromCtx(ctx context.Context) string {
	if userID, ok := ctx.Value(CtxKewJwtUserId).(string); ok {
		return userID
	}
	panic("must get user id from ctx fail")
}
