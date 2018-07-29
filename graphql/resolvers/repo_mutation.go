package resolvers

import (
	"context"
	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/graphql/models"
)

type repoMutationResolver struct{}

func (repoMutationResolver) NewBug(ctx context.Context, obj *models.RepositoryMutation, title string, message string) (bug.Snapshot, error) {
	b, err := obj.Repo.NewBug(title, message)
	if err != nil {
		return bug.Snapshot{}, err
	}

	snap := b.Snapshot()

	return *snap, nil
}
