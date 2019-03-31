package resolvers

import (
	"context"

	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/graphql/graph"
	"github.com/MichaelMure/git-bug/util/git"
)

var _ graph.MutationResolver = &mutationResolver{}

type mutationResolver struct {
	cache *cache.MultiRepoCache
}

func (r mutationResolver) getRepo(repoRef *string) (*cache.RepoCache, error) {
	if repoRef != nil {
		return r.cache.ResolveRepo(*repoRef)
	}

	return r.cache.DefaultRepo()
}

func (r mutationResolver) NewBug(ctx context.Context, repoRef *string, title string, message string, files []git.Hash) (bug.Snapshot, error) {
	repo, err := r.getRepo(repoRef)
	if err != nil {
		return bug.Snapshot{}, err
	}

	b, err := repo.NewBugWithFiles(title, message, files)
	if err != nil {
		return bug.Snapshot{}, err
	}

	snap := b.Snapshot()

	return *snap, nil
}

func (r mutationResolver) Commit(ctx context.Context, repoRef *string, prefix string) (bug.Snapshot, error) {
	repo, err := r.getRepo(repoRef)
	if err != nil {
		return bug.Snapshot{}, err
	}

	b, err := repo.ResolveBugPrefix(prefix)
	if err != nil {
		return bug.Snapshot{}, err
	}

	err = b.Commit()
	if err != nil {
		return bug.Snapshot{}, err
	}

	snap := b.Snapshot()

	return *snap, nil
}

func (r mutationResolver) AddComment(ctx context.Context, repoRef *string, prefix string, message string, files []git.Hash) (bug.Snapshot, error) {
	repo, err := r.getRepo(repoRef)
	if err != nil {
		return bug.Snapshot{}, err
	}

	b, err := repo.ResolveBugPrefix(prefix)
	if err != nil {
		return bug.Snapshot{}, err
	}

	_, err = b.AddCommentWithFiles(message, files)
	if err != nil {
		return bug.Snapshot{}, err
	}

	snap := b.Snapshot()

	return *snap, nil
}

func (r mutationResolver) ChangeLabels(ctx context.Context, repoRef *string, prefix string, added []string, removed []string) (bug.Snapshot, error) {
	repo, err := r.getRepo(repoRef)
	if err != nil {
		return bug.Snapshot{}, err
	}

	b, err := repo.ResolveBugPrefix(prefix)
	if err != nil {
		return bug.Snapshot{}, err
	}

	_, _, err = b.ChangeLabels(added, removed)
	if err != nil {
		return bug.Snapshot{}, err
	}

	snap := b.Snapshot()

	return *snap, nil
}

func (r mutationResolver) Open(ctx context.Context, repoRef *string, prefix string) (bug.Snapshot, error) {
	repo, err := r.getRepo(repoRef)
	if err != nil {
		return bug.Snapshot{}, err
	}

	b, err := repo.ResolveBugPrefix(prefix)
	if err != nil {
		return bug.Snapshot{}, err
	}

	_, err = b.Open()
	if err != nil {
		return bug.Snapshot{}, err
	}

	snap := b.Snapshot()

	return *snap, nil
}

func (r mutationResolver) Close(ctx context.Context, repoRef *string, prefix string) (bug.Snapshot, error) {
	repo, err := r.getRepo(repoRef)
	if err != nil {
		return bug.Snapshot{}, err
	}

	b, err := repo.ResolveBugPrefix(prefix)
	if err != nil {
		return bug.Snapshot{}, err
	}

	_, err = b.Close()
	if err != nil {
		return bug.Snapshot{}, err
	}

	snap := b.Snapshot()

	return *snap, nil
}

func (r mutationResolver) SetTitle(ctx context.Context, repoRef *string, prefix string, title string) (bug.Snapshot, error) {
	repo, err := r.getRepo(repoRef)
	if err != nil {
		return bug.Snapshot{}, err
	}

	b, err := repo.ResolveBugPrefix(prefix)
	if err != nil {
		return bug.Snapshot{}, err
	}

	_, err = b.SetTitle(title)
	if err != nil {
		return bug.Snapshot{}, err
	}

	snap := b.Snapshot()

	return *snap, nil
}
