package resolvers

import (
	"context"
	"time"

	"github.com/git-bug/git-bug/api/auth"
	"github.com/git-bug/git-bug/api/graphql/graph"
	"github.com/git-bug/git-bug/api/graphql/models"
	"github.com/git-bug/git-bug/cache"
	"github.com/git-bug/git-bug/entities/bug"
	"github.com/git-bug/git-bug/util/text"
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

	b, err := repo.Bugs().ResolvePrefix(bugPrefix)
	if err != nil {
		return nil, nil, err
	}
	return repo, b, nil
}

func (r mutationResolver) BugCreate(ctx context.Context, input models.BugCreateInput) (*models.BugCreatePayload, error) {
	repo, err := r.getRepo(input.RepoRef)
	if err != nil {
		return nil, err
	}

	author, err := auth.UserFromCtx(ctx, repo)
	if err != nil {
		return nil, err
	}

	b, op, err := repo.Bugs().NewRaw(author,
		time.Now().Unix(),
		text.CleanupOneLine(input.Title),
		text.Cleanup(input.Message),
		input.Files,
		nil)
	if err != nil {
		return nil, err
	}

	return &models.BugCreatePayload{
		ClientMutationID: input.ClientMutationID,
		Bug:              models.NewLoadedBug(b.Snapshot()),
		Operation:        op,
	}, nil
}

func (r mutationResolver) BugAddComment(ctx context.Context, input models.BugAddCommentInput) (*models.BugAddCommentPayload, error) {
	repo, b, err := r.getBug(input.RepoRef, input.Prefix)
	if err != nil {
		return nil, err
	}

	author, err := auth.UserFromCtx(ctx, repo)
	if err != nil {
		return nil, err
	}

	_, op, err := b.AddCommentRaw(author,
		time.Now().Unix(),
		text.Cleanup(input.Message),
		input.Files,
		nil)
	if err != nil {
		return nil, err
	}

	err = b.Commit()
	if err != nil {
		return nil, err
	}

	return &models.BugAddCommentPayload{
		ClientMutationID: input.ClientMutationID,
		Bug:              models.NewLoadedBug(b.Snapshot()),
		Operation:        op,
	}, nil
}

func (r mutationResolver) BugAddCommentAndClose(ctx context.Context, input models.BugAddCommentAndCloseInput) (*models.BugAddCommentAndClosePayload, error) {
	repo, b, err := r.getBug(input.RepoRef, input.Prefix)
	if err != nil {
		return nil, err
	}

	author, err := auth.UserFromCtx(ctx, repo)
	if err != nil {
		return nil, err
	}

	_, opAddComment, err := b.AddCommentRaw(author,
		time.Now().Unix(),
		text.Cleanup(input.Message),
		input.Files,
		nil)
	if err != nil {
		return nil, err
	}

	opClose, err := b.CloseRaw(author, time.Now().Unix(), nil)
	if err != nil {
		return nil, err
	}

	err = b.Commit()
	if err != nil {
		return nil, err
	}

	return &models.BugAddCommentAndClosePayload{
		ClientMutationID: input.ClientMutationID,
		Bug:              models.NewLoadedBug(b.Snapshot()),
		CommentOperation: opAddComment,
		StatusOperation:  opClose,
	}, nil
}

func (r mutationResolver) BugAddCommentAndReopen(ctx context.Context, input models.BugAddCommentAndReopenInput) (*models.BugAddCommentAndReopenPayload, error) {
	repo, b, err := r.getBug(input.RepoRef, input.Prefix)
	if err != nil {
		return nil, err
	}

	author, err := auth.UserFromCtx(ctx, repo)
	if err != nil {
		return nil, err
	}

	_, opAddComment, err := b.AddCommentRaw(author,
		time.Now().Unix(),
		text.Cleanup(input.Message),
		input.Files,
		nil)
	if err != nil {
		return nil, err
	}

	opReopen, err := b.OpenRaw(author, time.Now().Unix(), nil)
	if err != nil {
		return nil, err
	}

	err = b.Commit()
	if err != nil {
		return nil, err
	}

	return &models.BugAddCommentAndReopenPayload{
		ClientMutationID: input.ClientMutationID,
		Bug:              models.NewLoadedBug(b.Snapshot()),
		CommentOperation: opAddComment,
		StatusOperation:  opReopen,
	}, nil
}

func (r mutationResolver) BugEditComment(ctx context.Context, input models.BugEditCommentInput) (*models.BugEditCommentPayload, error) {
	repo, err := r.getRepo(input.RepoRef)
	if err != nil {
		return nil, err
	}

	b, target, err := repo.Bugs().ResolveComment(input.TargetPrefix)
	if err != nil {
		return nil, err
	}

	author, err := auth.UserFromCtx(ctx, repo)
	if err != nil {
		return nil, err
	}

	op, err := b.EditCommentRaw(
		author,
		time.Now().Unix(),
		target,
		text.Cleanup(input.Message),
		nil,
	)
	if err != nil {
		return nil, err
	}

	err = b.Commit()
	if err != nil {
		return nil, err
	}

	return &models.BugEditCommentPayload{
		ClientMutationID: input.ClientMutationID,
		Bug:              models.NewLoadedBug(b.Snapshot()),
		Operation:        op,
	}, nil
}

func (r mutationResolver) BugChangeLabels(ctx context.Context, input *models.BugChangeLabelInput) (*models.BugChangeLabelPayload, error) {
	repo, b, err := r.getBug(input.RepoRef, input.Prefix)
	if err != nil {
		return nil, err
	}

	author, err := auth.UserFromCtx(ctx, repo)
	if err != nil {
		return nil, err
	}

	results, op, err := b.ChangeLabelsRaw(
		author,
		time.Now().Unix(),
		text.CleanupOneLineArray(input.Added),
		text.CleanupOneLineArray(input.Removed),
		nil,
	)
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

	return &models.BugChangeLabelPayload{
		ClientMutationID: input.ClientMutationID,
		Bug:              models.NewLoadedBug(b.Snapshot()),
		Operation:        op,
		Results:          resultsPtr,
	}, nil
}

func (r mutationResolver) BugStatusOpen(ctx context.Context, input models.BugStatusOpenInput) (*models.BugStatusOpenPayload, error) {
	repo, b, err := r.getBug(input.RepoRef, input.Prefix)
	if err != nil {
		return nil, err
	}

	author, err := auth.UserFromCtx(ctx, repo)
	if err != nil {
		return nil, err
	}

	op, err := b.OpenRaw(author, time.Now().Unix(), nil)
	if err != nil {
		return nil, err
	}

	err = b.Commit()
	if err != nil {
		return nil, err
	}

	return &models.BugStatusOpenPayload{
		ClientMutationID: input.ClientMutationID,
		Bug:              models.NewLoadedBug(b.Snapshot()),
		Operation:        op,
	}, nil
}

func (r mutationResolver) BugStatusClose(ctx context.Context, input models.BugStatusCloseInput) (*models.BugStatusClosePayload, error) {
	repo, b, err := r.getBug(input.RepoRef, input.Prefix)
	if err != nil {
		return nil, err
	}

	author, err := auth.UserFromCtx(ctx, repo)
	if err != nil {
		return nil, err
	}

	op, err := b.CloseRaw(author, time.Now().Unix(), nil)
	if err != nil {
		return nil, err
	}

	err = b.Commit()
	if err != nil {
		return nil, err
	}

	return &models.BugStatusClosePayload{
		ClientMutationID: input.ClientMutationID,
		Bug:              models.NewLoadedBug(b.Snapshot()),
		Operation:        op,
	}, nil
}

func (r mutationResolver) BugSetTitle(ctx context.Context, input models.BugSetTitleInput) (*models.BugSetTitlePayload, error) {
	repo, b, err := r.getBug(input.RepoRef, input.Prefix)
	if err != nil {
		return nil, err
	}

	author, err := auth.UserFromCtx(ctx, repo)
	if err != nil {
		return nil, err
	}

	op, err := b.SetTitleRaw(
		author,
		time.Now().Unix(),
		text.CleanupOneLine(input.Title),
		nil,
	)
	if err != nil {
		return nil, err
	}

	err = b.Commit()
	if err != nil {
		return nil, err
	}

	return &models.BugSetTitlePayload{
		ClientMutationID: input.ClientMutationID,
		Bug:              models.NewLoadedBug(b.Snapshot()),
		Operation:        op,
	}, nil
}
