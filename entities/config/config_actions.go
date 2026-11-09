package config

import (
	"context"

	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/entity/dag"
	"github.com/git-bug/git-bug/repository"
)

// Fetch retrieve updates from a remote
// This does not change the local config
func Fetch(ctx context.Context, repo repository.Repo, remote string) (string, error) {
	return dag.Fetch(ctx, def, repo, remote)
}

// Push update a remote with the local changes
func Push(ctx context.Context, repo repository.Repo, remote string) (string, error) {
	return dag.Push(ctx, def, repo, remote)
}

// Pull will do a Fetch + MergeAll
// This function will return an error if a merge fail, including when the local and
// remote configs are different ones (entity.MergeStatusConflict).
// Note: an author is necessary for the case where a merge commit is created, as this commit will
// have an author and may be signed if a signing key is available.
func Pull(ctx context.Context, repo repository.Repo, resolvers entity.Resolvers, remote string, mergeAuthor identity.Interface) error {
	return dag.Pull(ctx, def, wrapper, repo, resolvers, remote, mergeAuthor)
}

// MergeAll will merge the remote config, see dag.MergeAll. Its result has an empty
// id, and entity.MergeStatusConflict when the local and remote configs are different
// ones.
// Note: an author is necessary for the case where a merge commit is created, as this commit will
// have an author and may be signed if a signing key is available.
func MergeAll(ctx context.Context, repo repository.Repo, resolvers entity.Resolvers, remote string, mergeAuthor identity.Interface) <-chan entity.MergeResult {
	return dag.MergeAll(ctx, def, wrapper, repo, resolvers, remote, mergeAuthor)
}

// RemoveAll will remove the local config.
// RemoveAll is idempotent.
func RemoveAll(ctx context.Context, repo repository.Repo) error {
	return dag.RemoveAll(ctx, def, repo)
}
