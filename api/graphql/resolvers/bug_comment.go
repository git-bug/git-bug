package resolvers

import (
	"context"

	"github.com/git-bug/git-bug/api/graphql/graph"
	"github.com/git-bug/git-bug/api/graphql/models"
	"github.com/git-bug/git-bug/entities/bug"
)

var _ graph.BugCommentResolver = &commentResolver{}

type commentResolver struct{}

func (c commentResolver) Author(_ context.Context, obj *bug.Comment) (models.IdentityWrapper, error) {
	return models.NewLoadedIdentity(obj.Author), nil
}
