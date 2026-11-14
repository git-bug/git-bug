package cache

import (
	"context"

	"github.com/git-bug/git-bug/entities/config"
	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/entity/dag"
	"github.com/git-bug/git-bug/repository"
)

// sharedConfig is the single loaded copy of the repository config, see
// sharedEntity and refTracked.
type sharedConfig = sharedEntity[*config.Snapshot, config.Operation]

var _ cacheMgmt = &RepoCacheConfig{}
var _ dag.Tracked[*config.Snapshot, config.Operation] = &refTracked[*config.Snapshot, config.Operation]{}

// RepoCacheConfig holds the repository config, the settings shared by every clone
// of the repository, see config.Config.
type RepoCacheConfig struct {
	*SingletonCache[*config.Snapshot, config.Operation, *ConfigCache]
}

func NewRepoCacheConfig(repo repository.Repo,
	resolvers func() entity.Resolvers,
	getUserIdentity getUserIdentityFunc) *RepoCacheConfig {

	makeView := func(shared *sharedConfig, onCommit func()) *ConfigCache {
		return newConfigCache(shared, repo, resolvers, getUserIdentity, onCommit)
	}

	actions := SingletonActions[*config.Snapshot, config.Operation]{
		New: func() dag.Tracked[*config.Snapshot, config.Operation] {
			return config.NewConfig()
		},
		Head: config.Head,
		ReadWithResolver: func(ctx context.Context, repo repository.Repo, resolvers entity.Resolvers) (dag.Tracked[*config.Snapshot, config.Operation], error) {
			c, err := config.ReadWithResolver(ctx, repo, resolvers)
			if err != nil {
				return nil, err
			}
			return c, nil
		},
		NewCreate: func(repo repository.RepoClock, author identity.Interface, unixTime int64) (config.Operation, error) {
			return config.NewCreateOp(repo, author, unixTime)
		},
		RemoveAll: config.RemoveAll,
		MergeAll: func(ctx context.Context, repo repository.Repo, resolvers entity.Resolvers, remote string, mergeAuthor identity.Interface) <-chan entity.MergeResult {
			return config.MergeAll(ctx, repo, resolvers, remote, mergeAuthor)
		},
		EnsureClocks: config.EnsureClocks,
	}

	sc := NewSingletonCache[*config.Snapshot, config.Operation, *ConfigCache](
		repo, resolvers, getUserIdentity,
		makeView, actions,
		config.Typename, config.Namespace,
	)

	return &RepoCacheConfig{SingletonCache: sc}
}
