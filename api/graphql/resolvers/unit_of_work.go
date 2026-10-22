package resolvers

import (
	"context"
	"fmt"

	"github.com/99designs/gqlgen/graphql"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/gqlerror"

	"github.com/git-bug/git-bug/cache"
	"github.com/git-bug/git-bug/entity"
)

var _ graphql.HandlerExtension = CommitMutations{}
var _ graphql.ResponseInterceptor = CommitMutations{}

// CommitMutations is a gqlgen extension making the mutations of a request
// all-or-nothing: they only stage their operations, committed once they all
// succeeded. The operations staged on the same entity end up in a single operation
// pack per author.
//
// The atomicity is per entity: when a request changes several entities and one of
// them fails to commit, the ones committed before stay.
type CommitMutations struct{}

func (CommitMutations) ExtensionName() string {
	return "commit mutations"
}

func (CommitMutations) Validate(graphql.ExecutableSchema) error {
	return nil
}

func (CommitMutations) InterceptResponse(ctx context.Context, next graphql.ResponseHandler) *graphql.Response {
	// requests failing before execution (malformed body, parse error, unknown
	// operation) reach here too, without an operation to look at
	if !graphql.HasOperationContext(ctx) {
		return next(ctx)
	}
	op := graphql.GetOperationContext(ctx).Operation
	if op == nil || op.Operation != ast.Mutation {
		return next(ctx)
	}

	uow := &unitOfWork{bugs: make(map[*cache.RepoCache]map[entity.Id]*cache.BugCache)}
	resp := next(context.WithValue(ctx, unitOfWorkCtxKey{}, uow))
	if resp == nil {
		return nil
	}

	if len(resp.Errors) == 0 {
		err := uow.commit(ctx)
		if err != nil {
			resp.Errors = append(resp.Errors, gqlerror.Errorf("%s", err))
		}
	}
	// the payloads show the staged operations: with nothing committed, they
	// would be lying
	if len(resp.Errors) > 0 {
		resp.Data = nil
	}
	return resp
}

// unitOfWorkCtxKey is the context key of the unit of work, a type of its own to
// not collide with any other key.
type unitOfWorkCtxKey struct{}

// unitOfWork holds the views the mutations of a request stage their operations
// on, one per entity, for CommitMutations to commit them together. Mutation
// fields execute one after the other, so it's never used concurrently.
type unitOfWork struct {
	// the views, per repository then per bug: the same bug can be in several
	// repositories, each with its own view
	bugs map[*cache.RepoCache]map[entity.Id]*cache.BugCache
}

func unitOfWorkFromCtx(ctx context.Context) *unitOfWork {
	uow, ok := ctx.Value(unitOfWorkCtxKey{}).(*unitOfWork)
	if !ok {
		panic("mutation outside of a unit of work, see CommitMutations")
	}
	return uow
}

// commit commits the entities, stopping at the first failure.
func (uow *unitOfWork) commit(ctx context.Context) error {
	for _, bugs := range uow.bugs {
		for id, b := range bugs {
			err := b.CommitAsNeeded(ctx)
			if err != nil {
				return fmt.Errorf("can't commit bug %s: %w", id.Human(), err)
			}
		}
	}
	return nil
}

// bug returns the view of the bug the request stages on, b itself if it's the
// first one.
func (uow *unitOfWork) bug(repo *cache.RepoCache, b *cache.BugCache) *cache.BugCache {
	if existing, ok := uow.bugs[repo][b.Id()]; ok {
		return existing
	}
	if uow.bugs[repo] == nil {
		uow.bugs[repo] = make(map[entity.Id]*cache.BugCache)
	}
	uow.bugs[repo][b.Id()] = b
	return b
}

// resolveBug finds an existing bug. The bugs created by the request are not
// among them: their ids can't be known by the request.
func (uow *unitOfWork) resolveBug(ctx context.Context, repo *cache.RepoCache, prefix string) (*cache.BugCache, error) {
	b, err := repo.Bugs().ResolvePrefix(ctx, prefix)
	if err != nil {
		return nil, err
	}
	return uow.bug(repo, b), nil
}

func (uow *unitOfWork) resolveBugComment(ctx context.Context, repo *cache.RepoCache, prefix string) (*cache.BugCache, entity.CombinedId, error) {
	b, id, err := repo.Bugs().ResolveComment(ctx, prefix)
	if err != nil {
		return nil, id, err
	}
	return uow.bug(repo, b), id, nil
}
