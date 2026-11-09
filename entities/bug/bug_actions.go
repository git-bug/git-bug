package bug

import (
	"context"
	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/entity/dag"
	"github.com/git-bug/git-bug/repository"
)

// Fetch retrieve updates from a remote
// This does not change the local bugs state
func Fetch(ctx context.Context, repo repository.Repo, remote string) (string, error) {
	return dag.Fetch(ctx, def, repo, remote)
}

// Push update a remote with the local changes
func Push(ctx context.Context, repo repository.Repo, remote string) (string, error) {
	return dag.Push(ctx, def, repo, remote)
}

// Pull will do a Fetch + MergeAll
// This function will return an error if a merge fail
// Note: an author is necessary for the case where a merge commit is created, as this commit will
// have an author and may be signed if a signing key is available.
func Pull(ctx context.Context, repo repository.Repo, resolvers entity.Resolvers, remote string, mergeAuthor identity.Interface) error {
	return dag.Pull(ctx, def, wrapper, repo, resolvers, remote, mergeAuthor)
}

// MergeAll will merge all the available remote bug
// Note: an author is necessary for the case where a merge commit is created, as this commit will
// have an author and may be signed if a signing key is available.
func MergeAll(ctx context.Context, repo repository.Repo, resolvers entity.Resolvers, remote string, mergeAuthor identity.Interface) <-chan entity.MergeResult {
	return dag.MergeAll(ctx, def, wrapper, repo, resolvers, remote, mergeAuthor)
}

// ListHeads returns every local bug id, with the commit its reference points to.
func ListHeads(ctx context.Context, repo repository.Repo) (map[entity.Id]repository.Hash, error) {
	return dag.ListHeads(ctx, def, repo)
}

// Head returns the commit the reference of a local bug points to, or
// repository.ErrNotFound if there is none.
func Head(ctx context.Context, repo repository.Repo, id entity.Id) (repository.Hash, error) {
	return dag.Head(ctx, def, repo, id)
}

// IdsFromKeys returns the ids of the bugs stored in the references with these keys,
// ignoring the keys that can't hold a bug.
func IdsFromKeys(keys []string) []entity.Id {
	return dag.IdsFromKeys(def, keys)
}

// Remove will remove a local bug from its entity.Id
func Remove(ctx context.Context, repo repository.Repo, id entity.Id) error {
	return dag.Remove(ctx, def, repo, id)
}

// RemoveAll will remove all local bugs.
// RemoveAll is idempotent.
func RemoveAll(ctx context.Context, repo repository.Repo) error {
	return dag.RemoveAll(ctx, def, repo)
}
