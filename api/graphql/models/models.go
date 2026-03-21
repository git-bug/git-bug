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

// GitCommitMeta is a wrapper around a CommitMeta that includes the Repo,
// to keep the repo context in sub-resolvers.
type GitCommitMeta struct {
	Repo *cache.RepoCache
	repository.CommitMeta
}
