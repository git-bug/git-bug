// Package resolvers contains the various GraphQL resolvers
package resolvers

import (
	"github.com/git-bug/git-bug/api/graphql/graph"
	"github.com/git-bug/git-bug/cache"
)

var _ graph.ResolverRoot = &RootResolver{}

type RootResolver struct {
	*cache.MultiRepoCache
}

func NewRootResolver(mrc *cache.MultiRepoCache) *RootResolver {
	return &RootResolver{
		MultiRepoCache: mrc,
	}
}

func (r RootResolver) Query() graph.QueryResolver {
	return &rootQueryResolver{
		cache: r.MultiRepoCache,
	}
}

func (r RootResolver) Mutation() graph.MutationResolver {
	return &mutationResolver{
		cache: r.MultiRepoCache,
	}
}

func (RootResolver) Repository() graph.RepositoryResolver {
	return &repoResolver{}
}

func (RootResolver) Bug() graph.BugResolver {
	return &bugResolver{}
}

func (RootResolver) Color() graph.ColorResolver {
	return &colorResolver{}
}

func (r RootResolver) Comment() graph.CommentResolver {
	return &commentResolver{}
}

func (RootResolver) Label() graph.LabelResolver {
	return &labelResolver{}
}

func (r RootResolver) Identity() graph.IdentityResolver {
	return &identityResolver{}
}

func (RootResolver) CommentHistoryStep() graph.CommentHistoryStepResolver {
	return &commentHistoryStepResolver{}
}

func (RootResolver) AddCommentTimelineItem() graph.AddCommentTimelineItemResolver {
	return &addCommentTimelineItemResolver{}
}

func (RootResolver) CreateTimelineItem() graph.CreateTimelineItemResolver {
	return &createTimelineItemResolver{}
}

func (r RootResolver) LabelChangeTimelineItem() graph.LabelChangeTimelineItemResolver {
	return &labelChangeTimelineItem{}
}

func (r RootResolver) SetStatusTimelineItem() graph.SetStatusTimelineItemResolver {
	return &setStatusTimelineItem{}
}

func (r RootResolver) SetTitleTimelineItem() graph.SetTitleTimelineItemResolver {
	return &setTitleTimelineItem{}
}

func (RootResolver) CreateOperation() graph.CreateOperationResolver {
	return &createOperationResolver{}
}

func (RootResolver) AddCommentOperation() graph.AddCommentOperationResolver {
	return &addCommentOperationResolver{}
}

func (r RootResolver) EditCommentOperation() graph.EditCommentOperationResolver {
	return &editCommentOperationResolver{}
}

func (RootResolver) LabelChangeOperation() graph.LabelChangeOperationResolver {
	return &labelChangeOperationResolver{}
}

func (RootResolver) SetStatusOperation() graph.SetStatusOperationResolver {
	return &setStatusOperationResolver{}
}

func (RootResolver) SetTitleOperation() graph.SetTitleOperationResolver {
	return &setTitleOperationResolver{}
}
