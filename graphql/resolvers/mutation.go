package resolvers

import (
	"context"

	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/cache"
)

type mutationResolver struct {
	cache cache.Cacher
}

func (r mutationResolver) getRepo(repoRef *string) (cache.RepoCacher, error) {
	if repoRef != nil {
		return r.cache.ResolveRepo(*repoRef)
	}

	return r.cache.DefaultRepo()
}

func (r mutationResolver) NewBug(ctx context.Context, repoRef *string, title string, message string) (bug.Snapshot, error) {
	repo, err := r.getRepo(repoRef)
	if err != nil {
		return bug.Snapshot{}, err
	}

	b, err := repo.NewBug(title, message)
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

	err = repo.Commit(b)
	if err != nil {
		return bug.Snapshot{}, err
	}

	snap := b.Snapshot()

	return *snap, nil
}

func (r mutationResolver) AddComment(ctx context.Context, repoRef *string, prefix string, message string) (bug.Snapshot, error) {
	repo, err := r.getRepo(repoRef)
	if err != nil {
		return bug.Snapshot{}, err
	}

	b, err := repo.AddComment(repoRef, prefix, message)
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

	b, err := repo.ChangeLabels(repoRef, prefix, added, removed)
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

	b, err := repo.Open(repoRef, prefix)
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

	b, err := repo.Close(repoRef, prefix)
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

	b, err := repo.SetTitle(repoRef, prefix, title)
	if err != nil {
		return bug.Snapshot{}, err
	}

	snap := b.Snapshot()

	return *snap, nil
}
