package cache

import (
	"context"
	"time"

	"github.com/git-bug/git-bug/entities/common"
	"github.com/git-bug/git-bug/entities/config"
	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/repository"
)

var _ config.Mutable = &ConfigCache{}

// ConfigCache is a view of the repository config loaded in the cache, see
// CachedEntityBase: what it stages is its own until committed.
//
// There is at most one config: the first commit with no config creates it. If
// another one got created meanwhile, the change is committed on top of that one
// instead, see refTracked.
type ConfigCache struct {
	CachedEntityBase[*config.Snapshot, config.Operation]
}

func newConfigCache(shared *sharedConfig, repo repository.Repo, resolvers func() entity.Resolvers, getUserIdentity getUserIdentityFunc, onCommit func()) *ConfigCache {
	return &ConfigCache{
		CachedEntityBase: CachedEntityBase[*config.Snapshot, config.Operation]{
			repo:            repo,
			resolvers:       resolvers,
			onCommit:        onCommit,
			getUserIdentity: getUserIdentity,
			shared:          shared,
		},
	}
}

// Id returns the id of the config, unset until it exists: it is the id of the
// creation, which is only made by the first commit, see refTracked.
func (c *ConfigCache) Id() entity.Id {
	return c.shared.Id()
}

// withUser calls f with the user identity as author, at the current time.
func withUser[T any](ctx context.Context, c *ConfigCache, f func(author identity.Interface, unixTime int64) (T, error)) (T, error) {
	author, err := c.getUserIdentity(ctx)
	if err != nil {
		return *new(T), err
	}
	return f(author, time.Now().Unix())
}

func (c *ConfigCache) RepoSetDescription(ctx context.Context, description string) (*config.RepoSetDescriptionOperation, error) {
	return withUser(ctx, c, func(author identity.Interface, unixTime int64) (*config.RepoSetDescriptionOperation, error) {
		return c.RepoSetDescriptionRaw(author, unixTime, description)
	})
}

func (c *ConfigCache) RepoSetDescriptionRaw(author identity.Interface, unixTime int64, description string) (*config.RepoSetDescriptionOperation, error) {
	return config.RepoSetDescription(c, c.repo, author, unixTime, description)
}

func (c *ConfigCache) LabelsSet(ctx context.Context, name common.Label, change config.LabelsDefinition) (*config.LabelsSetOperation, error) {
	return withUser(ctx, c, func(author identity.Interface, unixTime int64) (*config.LabelsSetOperation, error) {
		return c.LabelsSetRaw(author, unixTime, name, change)
	})
}

func (c *ConfigCache) LabelsSetRaw(author identity.Interface, unixTime int64, name common.Label, change config.LabelsDefinition) (*config.LabelsSetOperation, error) {
	return config.LabelsSet(c, c.repo, author, unixTime, name, change)
}

func (c *ConfigCache) LabelsRemove(ctx context.Context, name common.Label) (*config.LabelsRemoveOperation, error) {
	return withUser(ctx, c, func(author identity.Interface, unixTime int64) (*config.LabelsRemoveOperation, error) {
		return c.LabelsRemoveRaw(author, unixTime, name)
	})
}

func (c *ConfigCache) LabelsRemoveRaw(author identity.Interface, unixTime int64, name common.Label) (*config.LabelsRemoveOperation, error) {
	return config.LabelsRemove(c, c.repo, author, unixTime, name)
}

func (c *ConfigCache) LabelsAllow(ctx context.Context, name common.Label) (*config.LabelsAllowOperation, error) {
	return withUser(ctx, c, func(author identity.Interface, unixTime int64) (*config.LabelsAllowOperation, error) {
		return c.LabelsAllowRaw(author, unixTime, name)
	})
}

func (c *ConfigCache) LabelsAllowRaw(author identity.Interface, unixTime int64, name common.Label) (*config.LabelsAllowOperation, error) {
	return config.LabelsAllow(c, c.repo, author, unixTime, name)
}

func (c *ConfigCache) LabelsDisallow(ctx context.Context, name common.Label) (*config.LabelsDisallowOperation, error) {
	return withUser(ctx, c, func(author identity.Interface, unixTime int64) (*config.LabelsDisallowOperation, error) {
		return c.LabelsDisallowRaw(author, unixTime, name)
	})
}

func (c *ConfigCache) LabelsDisallowRaw(author identity.Interface, unixTime int64, name common.Label) (*config.LabelsDisallowOperation, error) {
	return config.LabelsDisallow(c, c.repo, author, unixTime, name)
}

func (c *ConfigCache) LabelsRestrict(ctx context.Context) ([]*config.LabelsAllowOperation, error) {
	return withUser(ctx, c, func(author identity.Interface, unixTime int64) ([]*config.LabelsAllowOperation, error) {
		return c.LabelsRestrictRaw(author, unixTime)
	})
}

func (c *ConfigCache) LabelsRestrictRaw(author identity.Interface, unixTime int64) ([]*config.LabelsAllowOperation, error) {
	return config.LabelsRestrict(c, c.repo, author, unixTime)
}

func (c *ConfigCache) LabelsLiftRestriction(ctx context.Context) ([]*config.LabelsDisallowOperation, error) {
	return withUser(ctx, c, func(author identity.Interface, unixTime int64) ([]*config.LabelsDisallowOperation, error) {
		return c.LabelsLiftRestrictionRaw(author, unixTime)
	})
}

func (c *ConfigCache) LabelsLiftRestrictionRaw(author identity.Interface, unixTime int64) ([]*config.LabelsDisallowOperation, error) {
	return config.LabelsLiftRestriction(c, c.repo, author, unixTime)
}
