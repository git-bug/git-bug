package models

import (
	"github.com/MichaelMure/git-bug/cache"
)

type Repository struct {
	Cache cache.Cacher
	Repo  cache.RepoCacher
}

type RepositoryMutation struct {
	Cache cache.Cacher
	Repo  cache.RepoCacher
}
