package cache

import (
	"context"

	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/repository"
)

type RepoCacheIdentity struct {
	*SubCache[*identity.Identity, *IdentityExcerpt, *IdentityCache]
}

func NewRepoCacheIdentity(repo repository.Repo,
	resolvers func() entity.Resolvers,
	getUserIdentity getUserIdentityFunc) *RepoCacheIdentity {

	makeView := func(i *identity.Identity, onCommit func() error) *IdentityCache {
		// a copy, so that what the view mutates stays its own until committed
		return NewIdentityCache(i.Clone(), repo, onCommit)
	}

	makeIndex := func(i *identity.Identity) []string {
		// no indexing
		return nil
	}

	// TODO: this is terribly ugly, but we are currently stuck with the fact that identities are NOT using the fancy dag framework.
	//   This lead to various complication here and there to handle entities generically, and avoid large code duplication.
	//   TL;DR: something has to give, and this is the less ugly solution I found. This "normalize" identities as just another "dag framework"
	//   entity. Ideally identities would be converted to the dag framework, but right now that could lead to potential attack: if an old
	//   private key is leaked, it would be possible to craft a legal identity update that take over the most recent version. While this is
	//   meaningless in the case of a normal entity, it's really an issues for identities.
	//
	//   Until then, identities get a simpler version of the write model of the other entities: an IdentityCache is a view
	//   over a copy of the loaded identity, taken when it is resolved, that it mutates and commits on its own. The loaded
	//   identity holds committed state only. A refresh replaces it rather than updating it in place, so a view, or a bug,
	//   resolved before keeps the previous copy until it is resolved again.

	actions := Actions[*identity.Identity]{
		ReadWithResolver: func(ctx context.Context, repo repository.Repo, resolvers entity.Resolvers, id entity.Id) (*identity.Identity, error) {
			return identity.Read(ctx, repo, id)
		},
		Refresh: func(ctx context.Context, repo repository.Repo, resolvers entity.Resolvers, i *identity.Identity) (*identity.Identity, error) {
			return identity.Read(ctx, repo, i.Id())
		},
		Remove:    identity.Remove,
		RemoveAll: identity.RemoveAll,
		MergeAll: func(ctx context.Context, repo repository.Repo, resolvers entity.Resolvers, remote string, mergeAuthor identity.Interface) <-chan entity.MergeResult {
			return identity.MergeAll(ctx, repo, remote)
		},
		// identities carry no clock of their own, they record the others'
		EnsureClocks: func(repo repository.Repo) error { return nil },
	}

	sc := NewSubCache[*identity.Identity, *IdentityExcerpt, *IdentityCache](
		repo, resolvers, getUserIdentity,
		makeView, newIdentityExcerpt, makeIndex, actions,
		identity.Typename, identity.Namespace,
		formatVersion, defaultMaxLoadedBugs,
	)

	return &RepoCacheIdentity{SubCache: sc}
}

// ResolveIdentityImmutableMetadata retrieve an Identity that has the exact given metadata on
// one of its version. If multiple version have the same key, the first defined take precedence.
func (c *RepoCacheIdentity) ResolveIdentityImmutableMetadata(ctx context.Context, key string, value string) (*IdentityCache, error) {
	return c.ResolveMatcher(ctx, func(excerpt *IdentityExcerpt) bool {
		return excerpt.ImmutableMetadata[key] == value
	})
}

// New create a new identity
// The new identity is written in the repository (commit)
func (c *RepoCacheIdentity) New(name string, email string) (*IdentityCache, error) {
	return c.NewRaw(name, email, "", "", nil, nil)
}

// NewFull create a new identity
// The new identity is written in the repository (commit)
func (c *RepoCacheIdentity) NewFull(name string, email string, login string, avatarUrl string, keys []*identity.Key) (*IdentityCache, error) {
	return c.NewRaw(name, email, login, avatarUrl, keys, nil)
}

func (c *RepoCacheIdentity) NewRaw(name string, email string, login string, avatarUrl string, keys []*identity.Key, metadata map[string]string) (*IdentityCache, error) {
	i, err := identity.NewIdentityFull(c.repo, name, email, login, avatarUrl, keys)
	if err != nil {
		return nil, err
	}
	return c.finishIdentity(i, metadata)
}

func (c *RepoCacheIdentity) NewFromGitUser() (*IdentityCache, error) {
	return c.NewFromGitUserRaw(nil)
}

func (c *RepoCacheIdentity) NewFromGitUserRaw(metadata map[string]string) (*IdentityCache, error) {
	i, err := identity.NewFromGitUser(c.repo)
	if err != nil {
		return nil, err
	}
	return c.finishIdentity(i, metadata)
}

func (c *RepoCacheIdentity) finishIdentity(i *identity.Identity, metadata map[string]string) (*IdentityCache, error) {
	for key, value := range metadata {
		i.SetMetadata(key, value)
	}

	err := i.Commit(c.repo)
	if err != nil {
		return nil, err
	}

	err = c.onCommit(i)
	if err != nil {
		return nil, err
	}

	return c.newView(i), nil
}
