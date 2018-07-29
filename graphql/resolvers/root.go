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

func (r RootResolver) BugConnection() graph.BugConnectionResolver {
	panic("implement me")
}

func (r RootResolver) CommentConnection() graph.CommentConnectionResolver {
	return &createOperationResolver{}
}

func (r RootResolver) OperationConnection() graph.OperationConnectionResolver {
	panic("implement me")
}

func (r RootResolver) Query() graph.QueryResolver {
	return &rootQueryResolver{
		cache: &r.RootCache,
	}
}

func (r RootResolver) Bug() graph.BugResolver {
	return &bugResolver{
		cache: &r.RootCache,
	}
}

func (r RootResolver) Repository() graph.RepositoryResolver {
	return &repoResolver{
		cache: &r.RootCache,
	}
}
