package cache

import (
	"context"
	"errors"
	"maps"
	"slices"
	"sort"
	"time"

	"github.com/git-bug/git-bug/entities/bug"
	"github.com/git-bug/git-bug/entities/common"
	"github.com/git-bug/git-bug/entities/config"
	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/entity/dag"
	"github.com/git-bug/git-bug/query"
	"github.com/git-bug/git-bug/repository"
	"github.com/git-bug/git-bug/util/lamport"
)

// sharedBug is the single loaded copy of a bug, see sharedEntity. It is compiled
// with the repository config, see judgedBug, and compiled again when the config
// changes.
type sharedBug = sharedEntity[*bug.Snapshot, bug.Operation]

func newSharedBug(b *bug.Bug, cfg func() *config.Snapshot) *sharedBug {
	return &sharedBug{
		Tracked: judgedBug{Bug: b, config: cfg},
		depends: func() any { return cfg() },
	}
}

// judgedBug is a bug whose operations are judged by the repository config, see
// bug.Bug.CompileWith.
type judgedBug struct {
	*bug.Bug
	config func() *config.Snapshot
}

func (b judgedBug) Compile() *bug.Snapshot {
	return b.CompileWith(b.config())
}

// configEra returns the id of the era of the config cfg judging a bug of edit
// time editTime, see config.Snapshot.EraId: the snapshot of the bug only changes
// with it.
func configEra(cfg *config.Snapshot, editTime lamport.Time) string {
	return cfg.EraId(dag.EditClock(bug.Namespace), editTime)
}

type RepoCacheBug struct {
	*SubCache[*sharedBug, *BugExcerpt, *BugCache]

	// config returns the latest repository config, which judges the bugs
	config func() *config.Snapshot
}

// NewRepoCacheBug returns the subcache of the bugs, judged by the repository
// config that cfg returns, see judgedBug. After the config changed, the next
// sync of every bug rebuilds those it judges differently, see configEra.
func NewRepoCacheBug(repo repository.Repo,
	resolvers func() entity.Resolvers,
	getUserIdentity getUserIdentityFunc,
	cfg func() *config.Snapshot) *RepoCacheBug {

	makeView := func(shared *sharedBug, onCommit func()) *BugCache {
		return newBugCache(shared, repo, resolvers, getUserIdentity, onCommit)
	}

	makeIndexData := func(b *sharedBug) []string {
		snap := b.Compile()
		var res []string
		for _, comment := range snap.Comments {
			res = append(res, comment.Message)
		}
		res = append(res, snap.Title)
		return res
	}

	actions := Actions[*sharedBug]{
		ListHeads:   bug.ListHeads,
		Head:        bug.Head,
		IdsFromKeys: bug.IdsFromKeys,
		ReadWithResolver: func(ctx context.Context, repo repository.Repo, resolvers entity.Resolvers, id entity.Id) (*sharedBug, error) {
			b, err := bug.ReadWithResolver(ctx, repo, resolvers, id)
			if err != nil {
				return nil, err
			}
			return newSharedBug(b, cfg), nil
		},
		Refresh: func(ctx context.Context, repo repository.Repo, resolvers entity.Resolvers, shared *sharedBug) (*sharedBug, error) {
			return shared, shared.Repair(ctx, repo, resolvers)
		},
		Remove:       bug.Remove,
		RemoveAll:    bug.RemoveAll,
		MergeAll:     bug.MergeAll,
		EnsureClocks: bug.EnsureClocks,
	}

	// judged by another era of the config than when it was built
	stale := func(excerpt *BugExcerpt) bool {
		return excerpt.ConfigEra != configEra(cfg(), excerpt.EditLamportTime)
	}

	sc := NewSubCache[*sharedBug, *BugExcerpt, *BugCache](
		repo, resolvers, getUserIdentity,
		makeView, newBugExcerpt, makeIndexData, stale, actions,
		bug.Typename, bug.Namespace,
		formatVersion, defaultMaxLoadedBugs,
	)

	return &RepoCacheBug{SubCache: sc, config: cfg}
}

// ResolveBugCreateMetadata retrieve a bug that has the exact given metadata on
// its Create operation, that is, the first operation. It fails if multiple bugs
// match.
func (c *RepoCacheBug) ResolveBugCreateMetadata(ctx context.Context, key string, value string) (*BugCache, error) {
	return c.ResolveMatcher(ctx, func(excerpt *BugExcerpt) bool {
		return excerpt.CreateMetadata[key] == value
	})
}

// ResolveComment search for a Bug/Comment combination matching the merged
// bug/comment Id prefix. Returns the Bug containing the Comment and the Comment's
// Id.
func (c *RepoCacheBug) ResolveComment(ctx context.Context, prefix string) (*BugCache, entity.CombinedId, error) {
	bugPrefix, _ := entity.SeparateIds(prefix)
	bugCandidate := make([]entity.Id, 0, 5)

	// build a list of possible matching bugs
	c.muMaps.RLock()
	for _, excerpt := range c.excerpts {
		if excerpt.Id().HasPrefix(bugPrefix) {
			bugCandidate = append(bugCandidate, excerpt.Id())
		}
	}
	c.muMaps.RUnlock()

	matchingBugIds := make([]entity.Id, 0, 5)
	matchingCommentId := entity.UnsetCombinedId
	var matchingBug *BugCache

	// search for matching comments
	// searching every bug candidate allow for some collision with the bug prefix only,
	// before being refined with the full comment prefix
	for _, bugId := range bugCandidate {
		b, err := c.Resolve(ctx, bugId)
		if err != nil {
			return nil, entity.UnsetCombinedId, err
		}

		for _, comment := range b.Snapshot().Comments {
			if comment.CombinedId().HasPrefix(prefix) {
				matchingBugIds = append(matchingBugIds, bugId)
				matchingBug = b
				matchingCommentId = comment.CombinedId()
			}
		}
	}

	if len(matchingBugIds) > 1 {
		return nil, entity.UnsetCombinedId, entity.NewErrMultipleMatch("bug/comment", matchingBugIds)
	} else if len(matchingBugIds) == 0 {
		return nil, entity.UnsetCombinedId, errors.New("comment doesn't exist")
	}

	return matchingBug, matchingCommentId, nil
}

// Query return the id of all Bug matching the given Query
func (c *RepoCacheBug) Query(ctx context.Context, q *query.Query) ([]entity.Id, error) {
	if q == nil {
		return c.AllIds(), nil
	}

	matcher := compileMatcher(q.Filters)

	// The search and the filters run without muMaps, which a slow search
	// would hold for as long: a sync waiting for it would then hold back every
	// reader. Only the excerpts are taken under it; they are replaced, never
	// changed, so they can be read after.
	var hits []string
	if q.Search != nil {
		index, err := c.repo.GetIndex(c.namespace, c.version)
		if err != nil {
			return nil, err
		}

		hits, err = index.Search(ctx, q.Search)
		if err != nil {
			return nil, err
		}
	}

	var found []*BugExcerpt
	c.muMaps.RLock()
	if q.Search != nil {
		for _, hit := range hits {
			// the index can be ahead of the excerpts, see SubCache.syncEntitiesLocked
			if excerpt, ok := c.excerpts[entity.Id(hit)]; ok {
				found = append(found, excerpt)
			}
		}
	} else {
		found = slices.Collect(maps.Values(c.excerpts))
	}
	c.muMaps.RUnlock()

	var filtered []*BugExcerpt
	for _, excerpt := range found {
		if matcher.Match(ctx, excerpt, c.resolvers()) {
			filtered = append(filtered, excerpt)
		}
	}

	var sorter sort.Interface

	switch q.OrderBy {
	case query.OrderById:
		sorter = BugsById(filtered)
	case query.OrderByCreation:
		sorter = BugsByCreationTime(filtered)
	case query.OrderByEdit:
		sorter = BugsByEditTime(filtered)
	default:
		return nil, errors.New("missing sort type")
	}

	switch q.OrderDirection {
	case query.OrderAscending:
		// Nothing to do
	case query.OrderDescending:
		sorter = sort.Reverse(sorter)
	default:
		return nil, errors.New("missing sort direction")
	}

	sort.Sort(sorter)

	result := make([]entity.Id, len(filtered))

	for i, val := range filtered {
		result[i] = val.Id()
	}

	return result, nil
}

// ValidLabels returns the labels that may be added to a bug, sorted: the labels
// the config allows if it restricts them, otherwise the labels it defines and
// those already used.
func (c *RepoCacheBug) ValidLabels() []common.Label {
	labels := c.config().Labels
	if allowed := labels.AllowedNames(); len(allowed) > 0 {
		return allowed
	}

	set := map[common.Label]struct{}{}
	for _, l := range labels.Names() {
		set[l] = struct{}{}
	}

	c.muMaps.RLock()
	for _, excerpt := range c.excerpts {
		for _, l := range excerpt.Labels {
			set[l] = struct{}{}
		}
	}
	c.muMaps.RUnlock()

	return slices.Sorted(maps.Keys(set))
}

// New create a new bug
// The creation is staged, like any operation: Commit writes the new bug in the
// repository.
func (c *RepoCacheBug) New(ctx context.Context, title string, message string) (*BugCache, *bug.CreateOperation, error) {
	return c.NewWithFiles(ctx, title, message, nil)
}

// NewWithFiles create a new bug with attached files for the message
// The creation is staged, like any operation: Commit writes the new bug in the
// repository.
func (c *RepoCacheBug) NewWithFiles(ctx context.Context, title string, message string, files []repository.Hash) (*BugCache, *bug.CreateOperation, error) {
	author, err := c.getUserIdentity(ctx)
	if err != nil {
		return nil, nil, err
	}

	return c.NewRaw(author, time.Now().Unix(), title, message, files, nil)
}

// NewRaw create a new bug with attached files for the message, as
// well as metadata for the Create operation.
// The creation is staged, like any operation: Commit writes the new bug in the
// repository. Until then, the bug is only known to the returned view.
func (c *RepoCacheBug) NewRaw(author identity.Interface, unixTime int64, title string, message string, files []repository.Hash, metadata map[string]string) (*BugCache, *bug.CreateOperation, error) {
	op := bug.NewCreateOp(author, unixTime, title, message, files)
	for key, val := range metadata {
		op.SetMetadata(key, val)
	}
	if err := op.Validate(); err != nil {
		return nil, nil, err
	}

	cached := c.newView(newSharedBug(bug.NewBug(), c.config))
	cached.Append(op)
	return cached, op, nil
}
