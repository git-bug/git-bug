// Package resolvers contains the various GraphQL resolvers
package resolvers

import (
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/graphql/graph"
)

var _ graph.ResolverRoot = &RootResolver{}

type RootResolver struct {
	cache.MultiRepoCache
}

func NewRootResolver() *RootResolver {
	return &RootResolver{
		MultiRepoCache: cache.NewMultiRepoCache(),
	}
}

func (r RootResolver) Query() graph.QueryResolver {
	return &rootQueryResolver{
		cache: &r.MultiRepoCache,
	}
}

func (r RootResolver) Mutation() graph.MutationResolver {
	return &mutationResolver{
		cache: &r.MultiRepoCache,
	}
}

func (RootResolver) Bug() graph.BugResolver {
	return &bugResolver{}
}

func (RootResolver) Color() graph.ColorResolver {
	return &colorResolver{}
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
	return &labelChangeOperation{}
}

func (RootResolver) Repository() graph.RepositoryResolver {
	return &repoResolver{}
}

func (RootResolver) SetStatusOperation() graph.SetStatusOperationResolver {
	return &setStatusOperationResolver{}
}

func (RootResolver) SetTitleOperation() graph.SetTitleOperationResolver {
	return &setTitleOperationResolver{}
}
