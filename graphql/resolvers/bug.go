package resolvers

import (
	"context"
		"github.com/MichaelMure/git-bug/cache"
		"github.com/MichaelMure/git-bug/graphql/models"
)

type bugResolver struct {
	cache cache.Cacher
}

func (bugResolver) Comments(ctx context.Context, obj *models.Bug, input models.ConnectionInput) (models.CommentConnection, error) {
	panic("implement me")
}

func (bugResolver) Operations(ctx context.Context, obj *models.Bug, input models.ConnectionInput) (models.OperationConnection, error) {
	panic("implement me")
}


