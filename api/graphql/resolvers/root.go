// Package resolvers contains the various GraphQL resolvers
package resolvers

import (
	"github.com/git-bug/git-bug/api/graphql/broker"
	"github.com/git-bug/git-bug/api/graphql/graph"
	"github.com/git-bug/git-bug/cache"
)

var _ graph.ResolverRoot = &RootResolver{}

type RootResolver struct {
	*cache.MultiRepoCache
	bugRootSubResolver
	broker *broker.Broker
}

// NewRootResolver returns the resolvers of the repositories of mrc, which must
// all be added already: the subscriptions follow those only.
func NewRootResolver(mrc *cache.MultiRepoCache) *RootResolver {
	b := broker.New()
	// without filter, registering can't fail
	_ = mrc.RegisterObserver(b, "", "")
	return &RootResolver{
		MultiRepoCache: mrc,
		broker:         b,
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

func (r RootResolver) Subscription() graph.SubscriptionResolver {
	return &subscriptionResolver{
		cache:  r.MultiRepoCache,
		broker: r.broker,
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

func (r RootResolver) GitRef() graph.GitRefResolver {
	return &gitRefResolver{}
}

func (r RootResolver) GitCommit() graph.GitCommitResolver {
	return &gitCommitResolver{}
}

func (r RootResolver) GitTreeEntry() graph.GitTreeEntryResolver {
	return &gitTreeEntryResolver{}
}
