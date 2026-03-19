package auth

import "context"

type ctxKey int

const userIDKKey ctxKey = iota + 1

func WithUserID(ctx context.Context, userID uint64) context.Context {
	return context.WithValue(ctx, userIDKKey, userID)
}

func UserIDFromContext(ctx context.Context) (uint64, bool) {
	v := ctx.Value(userIDKKey)
	if v == nil {
		return 0, false
	}
	id, ok := v.(uint64)
	return id, ok
}
