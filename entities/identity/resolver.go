package identity

import (
	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/repository"
)

var _ entity.Resolver = &SimpleResolver{}

// SimpleResolver is a Resolver loading Identities directly from a Repo
type SimpleResolver struct {
	repo repository.Repo
}

func NewSimpleResolver(repo repository.Repo) *SimpleResolver {
	return &SimpleResolver{repo: repo}
}

func (r *SimpleResolver) Resolve(id entity.Id) (entity.Interface, error) {
	return ReadLocal(r.repo, id)
}

var _ entity.Resolver = &StubResolver{}

// StubResolver is a Resolver that doesn't load anything, only returning IdentityStub instances
type StubResolver struct{}

func NewStubResolver() *StubResolver {
	return &StubResolver{}
}

func (s *StubResolver) Resolve(id entity.Id) (entity.Interface, error) {
	return &IdentityStub{id: id}, nil
}
