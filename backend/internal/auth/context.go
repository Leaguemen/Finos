package auth

import "context"

type contextKey struct{}

// Principal is the authenticated identity extracted from a valid JWT.
type Principal struct {
	UserID int64
	Role   string
}

func withPrincipal(ctx context.Context, principal Principal) context.Context {
	return context.WithValue(ctx, contextKey{}, principal)
}

// PrincipalFromContext returns the authenticated identity for this request.
func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	principal, ok := ctx.Value(contextKey{}).(Principal)
	return principal, ok
}
