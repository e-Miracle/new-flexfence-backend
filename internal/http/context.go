package http

import "context"

type contextKey string

const (
	ctxBusinessClaims contextKey = "business_claims"
	ctxUserAuth       contextKey = "user_auth"
)

type BusinessAuth struct {
	UserID         string
	OrganizationID string
	Role           string
	Email          string
}

type UserAuth struct {
	UserID string
	Email  string
	Role   string
}

func withBusinessAuth(ctx context.Context, auth BusinessAuth) context.Context {
	return context.WithValue(ctx, ctxBusinessClaims, auth)
}

func businessAuthFromContext(ctx context.Context) (BusinessAuth, bool) {
	auth, ok := ctx.Value(ctxBusinessClaims).(BusinessAuth)
	return auth, ok
}

func withUserAuth(ctx context.Context, auth UserAuth) context.Context {
	return context.WithValue(ctx, ctxUserAuth, auth)
}

func userAuthFromContext(ctx context.Context) (UserAuth, bool) {
	auth, ok := ctx.Value(ctxUserAuth).(UserAuth)
	return auth, ok && auth.UserID != ""
}

func userIDFromContext(ctx context.Context) (string, bool) {
	auth, ok := userAuthFromContext(ctx)
	return auth.UserID, ok
}
