package resolvers

import (
	"context"
	"time"

	"github.com/git-bug/git-bug/api/graphql/graph"
	"github.com/git-bug/git-bug/api/graphql/models"
	"github.com/git-bug/git-bug/entities/bug"
)

var _ graph.ReviewResolver = reviewResolver{}

type reviewResolver struct{}

func (reviewResolver) Author(_ context.Context, obj *bug.Review) (models.IdentityWrapper, error) {
	return models.NewLoadedIdentity(obj.Author), nil
}

func (reviewResolver) CreatedAt(_ context.Context, obj *bug.Review) (*time.Time, error) {
	t := obj.CreatedAt.Time()
	return &t, nil
}

var _ graph.ReviewCommentResolver = reviewCommentResolver{}

type reviewCommentResolver struct{}

func (reviewCommentResolver) Author(_ context.Context, obj *bug.ReviewComment) (models.IdentityWrapper, error) {
	return models.NewLoadedIdentity(obj.Author), nil
}

func (reviewCommentResolver) CreatedAt(_ context.Context, obj *bug.ReviewComment) (*time.Time, error) {
	t := obj.CreatedAt.Time()
	return &t, nil
}
