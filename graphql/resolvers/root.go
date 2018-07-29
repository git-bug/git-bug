package resolvers

import (
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/graphql/graph"
)

type Backend struct {
	cache.RootCache
}

func NewBackend() *Backend {
	return &Backend{
		RootCache: cache.NewCache(),
	}
}

func (r Backend) Query() graph.QueryResolver {
	return &rootQueryResolver{
		cache: &r.RootCache,
	}
}

func (r Backend) Mutation() graph.MutationResolver {
	return &mutationResolver{
		cache: &r.RootCache,
	}
}

func (Backend) AddCommentOperation() graph.AddCommentOperationResolver {
	return &addCommentOperationResolver{}
}

func (r Backend) Bug() graph.BugResolver {
	return &bugResolver{}
}

func (Backend) CreateOperation() graph.CreateOperationResolver {
	return &createOperationResolver{}
}

func (Backend) LabelChangeOperation() graph.LabelChangeOperationResolver {
	return &labelChangeOperation{}
}

func (r Backend) Repository() graph.RepositoryResolver {
	return &repoResolver{}
}

func (Backend) SetStatusOperation() graph.SetStatusOperationResolver {
	return &setStatusOperationResolver{}
}

func (Backend) SetTitleOperation() graph.SetTitleOperationResolver {
	return &setTitleOperationResolver{}
}
