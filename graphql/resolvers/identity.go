package resolvers

import (
	"context"

	"github.com/MichaelMure/git-bug/graphql/graph"
	"github.com/MichaelMure/git-bug/identity"
)

var _ graph.IdentityResolver = &identityResolver{}

type identityResolver struct{}

func (identityResolver) ID(ctx context.Context, obj identity.Interface) (string, error) {
	return obj.Id().String(), nil
}
