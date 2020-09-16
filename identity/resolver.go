package identity

import (
	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/repository"
)

// Resolver define the interface of an Identity resolver, able to load
// an identity from, for example, a repo or a cache.
type Resolver interface {
	ResolveIdentity(id entity.Id) (Interface, error)
}

// SimpleResolver is a Resolver loading Identities directly from a Repo
type SimpleResolver struct {
	repo repository.Repo
}

func NewSimpleResolver(repo repository.Repo) *SimpleResolver {
	return &SimpleResolver{repo: repo}
}

func (r *SimpleResolver) ResolveIdentity(id entity.Id) (Interface, error) {
	return ReadLocal(r.repo, id)
}

// StubResolver is a Resolver that doesn't load anything, only returning IdentityStub instances
type StubResolver struct{}

func NewStubResolver() *StubResolver {
	return &StubResolver{}
}

func (s *StubResolver) ResolveIdentity(id entity.Id) (Interface, error) {
	return &IdentityStub{id: id}, nil
}
