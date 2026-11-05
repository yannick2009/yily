package middleware

import (
	"context"

	"github.com/yannick2009/yily/internal/domain/model"
	"github.com/yannick2009/yily/internal/domain/types"
	"github.com/yannick2009/yily/internal/service"
)

type authUserContextKey struct{}
type authSessionContextKey struct{}
type authPATContextKey struct{}

// WithAuthenticatedIdentity stores the authenticated User and Session in the request context.
func WithAuthenticatedIdentity(ctx context.Context, user *model.User, session *model.Session) context.Context {
	ctx = context.WithValue(ctx, authUserContextKey{}, user)
	ctx = context.WithValue(ctx, authSessionContextKey{}, session)
	return ctx
}

// WithAuthenticatedPAT stores the authenticated User and PersonalAccessToken in the request context.
func WithAuthenticatedPAT(ctx context.Context, user *model.User, pat *model.PersonalAccessToken) context.Context {
	ctx = context.WithValue(ctx, authUserContextKey{}, user)
	ctx = context.WithValue(ctx, authPATContextKey{}, pat)
	return ctx
}

// AuthenticatedUserFromContext retrieves the authenticated User injected by the auth middleware from the context.
func AuthenticatedUserFromContext(ctx context.Context) (*model.User, bool) {
	user, ok := ctx.Value(authUserContextKey{}).(*model.User)
	return user, ok && user != nil
}

// AuthenticatedSessionFromContext retrieves the authenticated Session injected by the auth middleware from the context.
func AuthenticatedSessionFromContext(ctx context.Context) (*model.Session, bool) {
	session, ok := ctx.Value(authSessionContextKey{}).(*model.Session)
	return session, ok && session != nil
}

// AuthenticatedPATFromContext retrieves the authenticated PersonalAccessToken injected by the auth middleware from the context.
func AuthenticatedPATFromContext(ctx context.Context) (*model.PersonalAccessToken, bool) {
	pat, ok := ctx.Value(authPATContextKey{}).(*model.PersonalAccessToken)
	return pat, ok && pat != nil
}

// AuthenticatedScopeFromContext returns the effective permission scope of the current request. A session (super admin UI) implicitly grants write access; a PAT grants its own stored scope.
func AuthenticatedScopeFromContext(ctx context.Context) (types.PATScope, bool) {
	if pat, ok := AuthenticatedPATFromContext(ctx); ok {
		return pat.Scope, true
	}
	if _, ok := AuthenticatedSessionFromContext(ctx); ok {
		return types.PATScopeWrite, true
	}
	return "", false
}

// ActorFromContext resolves the authenticated actor from the request context for audit attribution.
// A Bearer PAT takes precedence over a session user, so token activity is attributed to the token.
func ActorFromContext(ctx context.Context) service.AuditActor {
	if pat, ok := AuthenticatedPATFromContext(ctx); ok {
		return service.AuditActor{Type: types.ActorTypeToken, ID: pat.ID, Name: pat.Name}
	}
	if user, ok := AuthenticatedUserFromContext(ctx); ok {
		return service.AuditActor{Type: types.ActorTypeUser, ID: user.ID, Name: user.Name}
	}
	return service.AuditActor{Type: types.ActorTypeSystem}
}
