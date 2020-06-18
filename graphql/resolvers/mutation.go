package resolvers

import (
	"context"

	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/graphql/graph"
	"github.com/MichaelMure/git-bug/graphql/models"
	"github.com/vektah/gqlparser/gqlerror"
)

var _ graph.MutationResolver = &readonlyMutationResolver{}

type readonlyMutationResolver struct{}

func (readonlyMutationResolver) NewBug(_ context.Context, _ models.NewBugInput) (*models.NewBugPayload, error) {
	return nil, gqlerror.Errorf("readonly mode")
}
func (readonlyMutationResolver) AddComment(_ context.Context, input models.AddCommentInput) (*models.AddCommentPayload, error) {
	return nil, gqlerror.Errorf("readonly mode")
}
func (readonlyMutationResolver) ChangeLabels(_ context.Context, input *models.ChangeLabelInput) (*models.ChangeLabelPayload, error) {
	return nil, gqlerror.Errorf("readonly mode")
}
func (readonlyMutationResolver) OpenBug(_ context.Context, input models.OpenBugInput) (*models.OpenBugPayload, error) {
	return nil, gqlerror.Errorf("readonly mode")
}
func (readonlyMutationResolver) CloseBug(_ context.Context, input models.CloseBugInput) (*models.CloseBugPayload, error) {
	return nil, gqlerror.Errorf("readonly mode")
}
func (readonlyMutationResolver) SetTitle(_ context.Context, input models.SetTitleInput) (*models.SetTitlePayload, error) {
	return nil, gqlerror.Errorf("readonly mode")
}

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

func (r mutationResolver) getBug(repoRef *string, bugPrefix string) (*cache.BugCache, error) {
	repo, err := r.getRepo(repoRef)
	if err != nil {
		return nil, err
	}

	return repo.ResolveBugPrefix(bugPrefix)
}

func (r mutationResolver) NewBug(_ context.Context, input models.NewBugInput) (*models.NewBugPayload, error) {
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
		Bug:              models.NewLoadedBug(b.Snapshot()),
		Operation:        op,
	}, nil
}

func (r mutationResolver) AddComment(_ context.Context, input models.AddCommentInput) (*models.AddCommentPayload, error) {
	b, err := r.getBug(input.RepoRef, input.Prefix)
	if err != nil {
		return nil, err
	}

	op, err := b.AddCommentWithFiles(input.Message, input.Files)
	if err != nil {
		return nil, err
	}

	err = b.Commit()
	if err != nil {
		return nil, err
	}

	return &models.AddCommentPayload{
		ClientMutationID: input.ClientMutationID,
		Bug:              models.NewLoadedBug(b.Snapshot()),
		Operation:        op,
	}, nil
}

func (r mutationResolver) ChangeLabels(_ context.Context, input *models.ChangeLabelInput) (*models.ChangeLabelPayload, error) {
	b, err := r.getBug(input.RepoRef, input.Prefix)
	if err != nil {
		return nil, err
	}

	results, op, err := b.ChangeLabels(input.Added, input.Removed)
	if err != nil {
		return nil, err
	}

	err = b.Commit()
	if err != nil {
		return nil, err
	}

	resultsPtr := make([]*bug.LabelChangeResult, len(results))
	for i, result := range results {
		resultsPtr[i] = &result
	}

	return &models.ChangeLabelPayload{
		ClientMutationID: input.ClientMutationID,
		Bug:              models.NewLoadedBug(b.Snapshot()),
		Operation:        op,
		Results:          resultsPtr,
	}, nil
}

func (r mutationResolver) OpenBug(_ context.Context, input models.OpenBugInput) (*models.OpenBugPayload, error) {
	b, err := r.getBug(input.RepoRef, input.Prefix)
	if err != nil {
		return nil, err
	}

	op, err := b.Open()
	if err != nil {
		return nil, err
	}

	err = b.Commit()
	if err != nil {
		return nil, err
	}

	return &models.OpenBugPayload{
		ClientMutationID: input.ClientMutationID,
		Bug:              models.NewLoadedBug(b.Snapshot()),
		Operation:        op,
	}, nil
}

func (r mutationResolver) CloseBug(_ context.Context, input models.CloseBugInput) (*models.CloseBugPayload, error) {
	b, err := r.getBug(input.RepoRef, input.Prefix)
	if err != nil {
		return nil, err
	}

	op, err := b.Close()
	if err != nil {
		return nil, err
	}

	err = b.Commit()
	if err != nil {
		return nil, err
	}

	return &models.CloseBugPayload{
		ClientMutationID: input.ClientMutationID,
		Bug:              models.NewLoadedBug(b.Snapshot()),
		Operation:        op,
	}, nil
}

func (r mutationResolver) SetTitle(_ context.Context, input models.SetTitleInput) (*models.SetTitlePayload, error) {
	b, err := r.getBug(input.RepoRef, input.Prefix)
	if err != nil {
		return nil, err
	}

	op, err := b.SetTitle(input.Title)
	if err != nil {
		return nil, err
	}

	err = b.Commit()
	if err != nil {
		return nil, err
	}

	return &models.SetTitlePayload{
		ClientMutationID: input.ClientMutationID,
		Bug:              models.NewLoadedBug(b.Snapshot()),
		Operation:        op,
	}, nil
}
