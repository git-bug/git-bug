package bug

import (
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/repository"
)

var _ entity.Resolver = &SimpleResolver{}

// SimpleResolver is a Resolver loading Bugs directly from a Repo
type SimpleResolver struct {
	repo repository.ClockedRepo
}

func NewSimpleResolver(repo repository.ClockedRepo) *SimpleResolver {
	return &SimpleResolver{repo: repo}
}

func (r *SimpleResolver) Resolve(id entity.Id) (entity.Resolved, error) {
	return Read(r.repo, id)
}
