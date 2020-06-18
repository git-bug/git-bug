package identity

import (
	"context"

	"github.com/MichaelMure/git-bug/repository"
)

// identityCtxKey is a unique context key, accessible only in this struct.
type identityCtxKey struct {
	repo string
}

// AttachToContext attaches an Identity to a context.
func AttachToContext(ctx context.Context, r repository.RepoCommon, u *Identity) context.Context {
	return context.WithValue(ctx, identityCtxKey{r.GetPath()}, u)
}

// ForContext retrieves an Identity from the context, or nil if no Identity is present.
func ForContext(ctx context.Context, r repository.RepoCommon) *Identity {
	u, ok := ctx.Value(identityCtxKey{r.GetPath()}).(*Identity)
	if !ok {
		return nil
	}
	return u
}
