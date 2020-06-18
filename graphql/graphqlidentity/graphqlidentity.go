// Package graphqlidentity contains helpers for managing identities within the GraphQL API.
package graphqlidentity

import (
	"context"

	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/identity"
	"github.com/MichaelMure/git-bug/repository"
)

// identityCtxKey is a unique context key, accessible only in this package.
var identityCtxKey = &struct{}{}

// AttachToContext attaches an Identity to a context.
func AttachToContext(ctx context.Context, u *identity.Identity) context.Context {
	return context.WithValue(ctx, identityCtxKey, u.Id())
}

// ForContext retrieves an IdentityCache from the context, or nil if no identity is present.
// If an error occurs while resolving the identity (e.g. I/O error), then it will be returned.
func ForContext(ctx context.Context, r *cache.RepoCache) (*cache.IdentityCache, error) {
	id, ok := ctx.Value(identityCtxKey).(entity.Id)
	if !ok {
		return nil, nil
	}
	return r.ResolveIdentity(id)
}

// ForContextUncached retrieves an Identity from the context, or nil if no identity is present.
// If an error occurs while resolving the identity (e.g. I/O error), then it will be returned.
func ForContextUncached(ctx context.Context, repo repository.Repo) (*identity.Identity, error) {
	id, ok := ctx.Value(identityCtxKey).(entity.Id)
	if !ok {
		return nil, nil
	}
	return identity.ReadLocal(repo, id)
}
