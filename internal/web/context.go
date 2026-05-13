package web

import (
	"context"

	"github.com/alexanderzull/file-converter/internal/auth"
)

type contextKey string

const userContextKey contextKey = "currentUser"

func withUser(ctx context.Context, user auth.User) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

func CurrentUser(ctx context.Context) (auth.User, bool) {
	user, ok := ctx.Value(userContextKey).(auth.User)
	return user, ok
}
