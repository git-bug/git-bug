// Package models contains the various GraphQL data models
package models

import (
	"github.com/git-bug/git-bug/cache"
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
