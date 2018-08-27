package resolvers

import (
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/graphql/graph"
)

type RootResolver struct {
	cache.RootCache
}

func NewRootResolver() *RootResolver {
	return &RootResolver{
		RootCache: cache.NewCache(),
	}
}

func (r RootResolver) Query() graph.QueryResolver {
	return &rootQueryResolver{
		cache: &r.RootCache,
	}
}

func (r RootResolver) Mutation() graph.MutationResolver {
	return &mutationResolver{
		cache: &r.RootCache,
	}
}

func (RootResolver) AddCommentOperation() graph.AddCommentOperationResolver {
	return &addCommentOperationResolver{}
}

func (r RootResolver) Bug() graph.BugResolver {
	return &bugResolver{}
}

func (RootResolver) CreateOperation() graph.CreateOperationResolver {
	return &createOperationResolver{}
}

func (RootResolver) LabelChangeOperation() graph.LabelChangeOperationResolver {
	return &labelChangeOperation{}
}

func (r RootResolver) Repository() graph.RepositoryResolver {
	return &repoResolver{}
}

func (RootResolver) SetStatusOperation() graph.SetStatusOperationResolver {
	return &setStatusOperationResolver{}
}

func (RootResolver) SetTitleOperation() graph.SetTitleOperationResolver {
	return &setTitleOperationResolver{}
}
