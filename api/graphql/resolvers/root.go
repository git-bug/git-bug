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

func (RootResolver) Color() graph.ColorResolver {
	return &colorResolver{}
}

func (r RootResolver) Identity() graph.IdentityResolver {
	return &identityResolver{}
}

func (RootResolver) Label() graph.LabelResolver {
	return &labelResolver{}
}

func (RootResolver) Repository() graph.RepositoryResolver {
	return &repoResolver{}
}

func (RootResolver) Bug() graph.BugResolver {
	return &bugResolver{}
}

func (RootResolver) BugAddCommentOperation() graph.BugAddCommentOperationResolver {
	return &bugAddCommentOperationResolver{}
}

func (RootResolver) BugAddCommentTimelineItem() graph.BugAddCommentTimelineItemResolver {
	return &bugAddCommentTimelineItemResolver{}
}

func (r RootResolver) BugComment() graph.BugCommentResolver {
	return &commentResolver{}
}

func (RootResolver) BugCommentHistoryStep() graph.BugCommentHistoryStepResolver {
	return &bugCommentHistoryStepResolver{}
}

func (RootResolver) BugCreateOperation() graph.BugCreateOperationResolver {
	return &bugCreateOperationResolver{}
}

func (RootResolver) BugCreateTimelineItem() graph.BugCreateTimelineItemResolver {
	return &bugCreateTimelineItemResolver{}
}

func (r RootResolver) BugEditCommentOperation() graph.BugEditCommentOperationResolver {
	return &bugEditCommentOperationResolver{}
}

func (RootResolver) BugLabelChangeOperation() graph.BugLabelChangeOperationResolver {
	return &bugLabelChangeOperationResolver{}
}

func (r RootResolver) BugLabelChangeTimelineItem() graph.BugLabelChangeTimelineItemResolver {
	return &bugLabelChangeTimelineItem{}
}

func (RootResolver) BugSetStatusOperation() graph.BugSetStatusOperationResolver {
	return &bugSetStatusOperationResolver{}
}

func (r RootResolver) BugSetStatusTimelineItem() graph.BugSetStatusTimelineItemResolver {
	return &bugSetStatusTimelineItem{}
}

func (r RootResolver) BugSetTitleOperation() graph.BugSetTitleOperationResolver {
	return &bugSetTitleOperationResolver{}
}

func (r RootResolver) BugSetTitleTimelineItem() graph.BugSetTitleTimelineItemResolver {
	return &bugSetTitleTimelineItem{}
}
