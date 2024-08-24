package resolvers

import (
	"context"

	"github.com/git-bug/git-bug/api/graphql/graph"
	"github.com/git-bug/git-bug/api/graphql/models"
)

var _ graph.IdentityResolver = &identityResolver{}

type identityResolver struct{}

func (identityResolver) ID(ctx context.Context, obj models.IdentityWrapper) (string, error) {
	return obj.Id().String(), nil
}

func (r identityResolver) HumanID(ctx context.Context, obj models.IdentityWrapper) (string, error) {
	return obj.Id().Human(), nil

}
