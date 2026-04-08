// Package models contains the various GraphQL data models
package models

import (
	"github.com/git-bug/git-bug/cache"
	"github.com/git-bug/git-bug/repository"
)

type ConnectionInput struct {
	After  *string
	Before *string
	First  *int
	Last   *int
}

type Repository struct {
	Repo *cache.RepoCache
}

// GitRef is a wrapper around a RefMeta that includes the Repo,
// to keep the repo context in sub-resolvers.
type GitRef struct {
	Repo *cache.RepoCache
	repository.RefMeta
}

// GitCommitMeta is a wrapper around a CommitMeta that includes the Repo,
// to keep the repo context in sub-resolvers.
type GitCommitMeta struct {
	Repo *cache.RepoCache
	repository.CommitMeta
}

// GitTreeEntry wraps a TreeEntry with the repository context (Repo, Ref, Path)
// of the resolution to that tree. SiblingNames lists all entries in the same
// directory so that the first lastCommit resolver call walks history for the whole
// directory at once; subsequent sibling calls hit the cache.
type GitTreeEntry struct {
	Repo         *cache.RepoCache
	Ref          string
	Path         string
	SiblingNames []string
	repository.TreeEntry
}
