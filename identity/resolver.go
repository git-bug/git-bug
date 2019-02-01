package identity

import "github.com/MichaelMure/git-bug/repository"

// Resolver define the interface of an Identity resolver, able to load
// an identity from, for example, a repo or a cache.
type Resolver interface {
	ResolveIdentity(id string) (Interface, error)
}

// DefaultResolver is a Resolver loading Identities directly from a Repo
type SimpleResolver struct {
	repo repository.Repo
}

func NewSimpleResolver(repo repository.Repo) *SimpleResolver {
	return &SimpleResolver{repo: repo}
}

func (r *SimpleResolver) ResolveIdentity(id string) (Interface, error) {
	return Read(r.repo, id)
}
