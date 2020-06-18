package resolvers

import (
	"context"
	"time"

	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/graphql/graph"
	"github.com/MichaelMure/git-bug/graphql/graphqlidentity"
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

func (r mutationResolver) getBug(repoRef *string, bugPrefix string) (*cache.RepoCache, *cache.BugCache, error) {
	repo, err := r.getRepo(repoRef)
	if err != nil {
		return nil, nil, err
	}

	bug, err := repo.ResolveBugPrefix(bugPrefix)
	if err != nil {
		return nil, nil, err
	}
	return repo, bug, nil
}

func (r mutationResolver) NewBug(ctx context.Context, input models.NewBugInput) (*models.NewBugPayload, error) {
	repo, err := r.getRepo(input.RepoRef)
	if err != nil {
		return nil, err
	}

	id, err := graphqlidentity.ForContext(ctx, repo)
	if err != nil {
		return nil, err
	} else if id == nil {
		return nil, ErrNotAuthenticated
	}

	b, op, err := repo.NewBugRaw(id, time.Now().Unix(), input.Title, input.Message, input.Files, nil)
	if err != nil {
		return nil, err
	}

	return &models.NewBugPayload{
		ClientMutationID: input.ClientMutationID,
		Bug:              models.NewLoadedBug(b.Snapshot()),
		Operation:        op,
	}, nil
}

func (r mutationResolver) AddComment(ctx context.Context, input models.AddCommentInput) (*models.AddCommentPayload, error) {
	repo, b, err := r.getBug(input.RepoRef, input.Prefix)
	if err != nil {
		return nil, err
	}

	id, err := graphqlidentity.ForContext(ctx, repo)
	if err != nil {
		return nil, err
	} else if id == nil {
		return nil, ErrNotAuthenticated
	}

	op, err := b.AddCommentRaw(id, time.Now().Unix(), input.Message, input.Files, nil)
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

func (r mutationResolver) ChangeLabels(ctx context.Context, input *models.ChangeLabelInput) (*models.ChangeLabelPayload, error) {
	repo, b, err := r.getBug(input.RepoRef, input.Prefix)
	if err != nil {
		return nil, err
	}

	id, err := graphqlidentity.ForContext(ctx, repo)
	if err != nil {
		return nil, err
	} else if id == nil {
		return nil, ErrNotAuthenticated
	}

	results, op, err := b.ChangeLabelsRaw(id, time.Now().Unix(), input.Added, input.Removed, nil)
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

func (r mutationResolver) OpenBug(ctx context.Context, input models.OpenBugInput) (*models.OpenBugPayload, error) {
	repo, b, err := r.getBug(input.RepoRef, input.Prefix)
	if err != nil {
		return nil, err
	}

	id, err := graphqlidentity.ForContext(ctx, repo)
	if err != nil {
		return nil, err
	} else if id == nil {
		return nil, ErrNotAuthenticated
	}

	op, err := b.OpenRaw(id, time.Now().Unix(), nil)
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

func (r mutationResolver) CloseBug(ctx context.Context, input models.CloseBugInput) (*models.CloseBugPayload, error) {
	repo, b, err := r.getBug(input.RepoRef, input.Prefix)
	if err != nil {
		return nil, err
	}

	id, err := graphqlidentity.ForContext(ctx, repo)
	if err != nil {
		return nil, err
	} else if id == nil {
		return nil, ErrNotAuthenticated
	}

	op, err := b.CloseRaw(id, time.Now().Unix(), nil)
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

func (r mutationResolver) SetTitle(ctx context.Context, input models.SetTitleInput) (*models.SetTitlePayload, error) {
	repo, b, err := r.getBug(input.RepoRef, input.Prefix)
	if err != nil {
		return nil, err
	}

	id, err := graphqlidentity.ForContext(ctx, repo)
	if err != nil {
		return nil, err
	} else if id == nil {
		return nil, ErrNotAuthenticated
	}

	op, err := b.SetTitleRaw(id, time.Now().Unix(), input.Title, nil)
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
