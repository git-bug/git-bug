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
	Cache cache.Cacher
	Repo  cache.RepoCacher
}

type RepositoryMutation struct {
	Cache cache.Cacher
	Repo  cache.RepoCacher
}
