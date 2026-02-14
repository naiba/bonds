package dav

import "context"

type contextKey string

const (
	ctxUserID    contextKey = "dav_user_id"
	ctxAccountID contextKey = "dav_account_id"
)

func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, ctxUserID, userID)
}

func UserIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(ctxUserID).(string)
	return id
}

func WithAccountID(ctx context.Context, accountID string) context.Context {
	return context.WithValue(ctx, ctxAccountID, accountID)
}

func AccountIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(ctxAccountID).(string)
	return id
}
