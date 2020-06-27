// Package auth contains helpers for managing identities within the GraphQL API.
package auth

import (
	"context"

	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/entity"
)

// identityCtxKey is a unique context key, accessible only in this package.
var identityCtxKey = &struct{}{}

// CtxWithUser attaches an Identity to a context.
func CtxWithUser(ctx context.Context, userId entity.Id) context.Context {
	return context.WithValue(ctx, identityCtxKey, userId)
}

// UserFromCtx retrieves an IdentityCache from the context.
// If there is no identity in the context, ErrNotAuthenticated is returned.
// If an error occurs while resolving the identity (e.g. I/O error), then it will be returned.
func UserFromCtx(ctx context.Context, r *cache.RepoCache) (*cache.IdentityCache, error) {
	id, ok := ctx.Value(identityCtxKey).(entity.Id)
	if !ok {
		return nil, ErrNotAuthenticated
	}
	return r.ResolveIdentity(id)
}
