package resolvers

import (
	"context"

	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/graphql/graph"
	"github.com/MichaelMure/git-bug/graphql/models"
)

var _ graph.MutationResolver = &mutationResolver{}

type mutationResolver struct {
	cache *cache.MultiRepoCache
}

func (r mutationResolver) getRepo(ref *string) (*cache.RepoCache, error) {
	if ref != nil {
		return r.cache.ResolveRepo(*ref)
	}

	return r.cache.DefaultRepo()
}

func (r mutationResolver) NewBug(ctx context.Context, input models.NewBugInput) (*models.NewBugPayload, error) {
	repo, err := r.getRepo(input.RepoRef)
	if err != nil {
		return nil, err
	}

	b, op, err := repo.NewBugWithFiles(input.Title, input.Message, input.Files)
	if err != nil {
		return nil, err
	}

	return &models.NewBugPayload{
		ClientMutationID: input.ClientMutationID,
		Bug:              b.Snapshot(),
		Operation:        op,
	}, nil
}

func (r mutationResolver) AddComment(ctx context.Context, input models.AddCommentInput) (*models.AddCommentPayload, error) {
	repo, err := r.getRepo(input.RepoRef)
	if err != nil {
		return nil, err
	}

	b, err := repo.ResolveBugPrefix(input.Prefix)
	if err != nil {
		return nil, err
	}

	op, err := b.AddCommentWithFiles(input.Message, input.Files)
	if err != nil {
		return nil, err
	}

	return &models.AddCommentPayload{
		ClientMutationID: input.ClientMutationID,
		Bug:              b.Snapshot(),
		Operation:        op,
	}, nil
}

func (r mutationResolver) ChangeLabels(ctx context.Context, input *models.ChangeLabelInput) (*models.ChangeLabelPayload, error) {
	repo, err := r.getRepo(input.RepoRef)
	if err != nil {
		return nil, err
	}

	b, err := repo.ResolveBugPrefix(input.Prefix)
	if err != nil {
		return nil, err
	}

	results, op, err := b.ChangeLabels(input.Added, input.Removed)
	if err != nil {
		return nil, err
	}

	resultsPtr := make([]*bug.LabelChangeResult, len(results))
	for i, result := range results {
		resultsPtr[i] = &result
	}

	return &models.ChangeLabelPayload{
		ClientMutationID: input.ClientMutationID,
		Bug:              b.Snapshot(),
		Operation:        op,
		Results:          resultsPtr,
	}, nil
}

func (r mutationResolver) OpenBug(ctx context.Context, input models.OpenBugInput) (*models.OpenBugPayload, error) {
	repo, err := r.getRepo(input.RepoRef)
	if err != nil {
		return nil, err
	}

	b, err := repo.ResolveBugPrefix(input.Prefix)
	if err != nil {
		return nil, err
	}

	op, err := b.Open()
	if err != nil {
		return nil, err
	}

	return &models.OpenBugPayload{
		ClientMutationID: input.ClientMutationID,
		Bug:              b.Snapshot(),
		Operation:        op,
	}, nil
}

func (r mutationResolver) CloseBug(ctx context.Context, input models.CloseBugInput) (*models.CloseBugPayload, error) {
	repo, err := r.getRepo(input.RepoRef)
	if err != nil {
		return nil, err
	}

	b, err := repo.ResolveBugPrefix(input.Prefix)
	if err != nil {
		return nil, err
	}

	op, err := b.Close()
	if err != nil {
		return nil, err
	}

	return &models.CloseBugPayload{
		ClientMutationID: input.ClientMutationID,
		Bug:              b.Snapshot(),
		Operation:        op,
	}, nil
}

func (r mutationResolver) SetTitle(ctx context.Context, input models.SetTitleInput) (*models.SetTitlePayload, error) {
	repo, err := r.getRepo(input.RepoRef)
	if err != nil {
		return nil, err
	}

	b, err := repo.ResolveBugPrefix(input.Prefix)
	if err != nil {
		return nil, err
	}

	op, err := b.SetTitle(input.Title)
	if err != nil {
		return nil, err
	}

	return &models.SetTitlePayload{
		ClientMutationID: input.ClientMutationID,
		Bug:              b.Snapshot(),
		Operation:        op,
	}, nil
}

func (r mutationResolver) Commit(ctx context.Context, input models.CommitInput) (*models.CommitPayload, error) {
	repo, err := r.getRepo(input.RepoRef)
	if err != nil {
		return nil, err
	}

	b, err := repo.ResolveBugPrefix(input.Prefix)
	if err != nil {
		return nil, err
	}

	err = b.Commit()
	if err != nil {
		return nil, err
	}

	return &models.CommitPayload{
		ClientMutationID: input.ClientMutationID,
		Bug:              b.Snapshot(),
	}, nil
}

func (r mutationResolver) CommitAsNeeded(ctx context.Context, input models.CommitAsNeededInput) (*models.CommitAsNeededPayload, error) {
	repo, err := r.getRepo(input.RepoRef)
	if err != nil {
		return nil, err
	}

	b, err := repo.ResolveBugPrefix(input.Prefix)
	if err != nil {
		return nil, err
	}

	err = b.CommitAsNeeded()
	if err != nil {
		return nil, err
	}

	return &models.CommitAsNeededPayload{
		ClientMutationID: input.ClientMutationID,
		Bug:              b.Snapshot(),
	}, nil
}
