package resolvers

import (
	"context"

	"github.com/MichaelMure/git-bug/api/graphql/graph"
	"github.com/MichaelMure/git-bug/api/graphql/models"
	"github.com/MichaelMure/git-bug/bug"
)

var _ graph.CommentResolver = &commentResolver{}

type commentResolver struct{}

func (c commentResolver) Author(_ context.Context, obj *bug.Comment) (models.IdentityWrapper, error) {
	return models.NewLoadedIdentity(obj.Author), nil
}
