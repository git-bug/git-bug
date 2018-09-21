package github

import "github.com/MichaelMure/git-bug/cache"

type github struct {
}

func (*github) Configure() error {
	panic("implement me")
}

func (*github) ImportAll(repo *cache.RepoCache) error {
	panic("implement me")
}

func (*github) Import(repo *cache.RepoCache, id string) error {
	panic("implement me")
}
