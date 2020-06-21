// Package models contains the various GraphQL data models
package models

import (
	"github.com/MichaelMure/git-bug/cache"
)

type ConnectionInput struct {
	After  *string
	Before *string
	First  *int
	Last   *int
}

type Repository struct {
	Cache *cache.MultiRepoCache
	Repo  *cache.RepoCache
}

type RepositoryMutation struct {
	Cache *cache.MultiRepoCache
	Repo  *cache.RepoCache
}
