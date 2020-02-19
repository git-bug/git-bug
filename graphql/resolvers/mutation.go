package resolvers

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"sync"

	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/graphql/graph"
	"github.com/MichaelMure/git-bug/graphql/models"
)

const transactionIdLength = 16

var _ graph.MutationResolver = &mutationResolver{}

type mutationResolver struct {
	cache *cache.MultiRepoCache

	mu           sync.Mutex
	transactions map[string]transaction
}

type transaction struct {
	bug *cache.BugCache
	ops []bug.Operation
}

func (r mutationResolver) makeId() (string, error) {
	result := make([]byte, transactionIdLength)

	i := 0
	for {
		_, err := rand.Read(result)
		if err != nil {
			panic(err)
		}
		id := base64.StdEncoding.EncodeToString(result)
		if _, has := r.transactions[id]; !has {
			return id, nil
		}
		i++
		if i > 100 {
			return "", fmt.Errorf("can't generate a transaction ID")
		}
	}
}

func (r mutationResolver) StartTransaction(_ context.Context, input models.StartTransactionInput) (*models.StartTransactionPayload, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	b, err := r.getBug(input.RepoRef, input.Prefix)
	if err != nil {
		return nil, err
	}

	id, err := r.makeId()
	if err != nil {
		return nil, err
	}

	r.transactions[id] = transaction{
		bug: b,
		ops: make([]bug.Operation, 0, 8),
	}

	return &models.StartTransactionPayload{
		ClientMutationID: input.ClientMutationID,
		ID:               id,
	}, nil
}

func (r mutationResolver) Commit(_ context.Context, input models.CommitInput) (*models.CommitPayload, error) {
	panic("implement me")
}

func (r mutationResolver) Rollback(_ context.Context, input models.RollbackInput) (*models.RollbackPayload, error) {
	panic("implement me")
}

func (r mutationResolver) getRepo(ref *string) (*cache.RepoCache, error) {
	if ref != nil {
		return r.cache.ResolveRepo(*ref)
	}

	return r.cache.DefaultRepo()
}

func (r mutationResolver) getBug(repoRef *string, bugPrefix string) (*cache.BugCache, error) {
	repo, err := r.getRepo(repoRef)
	if err != nil {
		return nil, err
	}

	return repo.ResolveBugPrefix(bugPrefix)
}

func (r mutationResolver) NewBug(_ context.Context, input models.NewBugInput, txID *string) (*models.NewBugPayload, error) {
	repo, err := r.getRepo(input.RepoRef)
	if err != nil {
		return nil, err
	}

	b, op, err := repo.NewBugWithFiles(input.Title, input.Message, input.Files)
	if err != nil {
		return nil, err
	}

	return &models.NewBugPayload{
		ClientMutationID: input.ClientMutationID,
		Bug:              models.NewLoadedBug(b.Snapshot()),
		Operation:        op,
	}, nil
}

func (r mutationResolver) AddComment(_ context.Context, input models.AddCommentInput, txID *string) (*models.AddCommentPayload, error) {
	b, err := r.getBug(input.RepoRef, input.Prefix)
	if err != nil {
		return nil, err
	}

	var op *bug.AddCommentOperation
	if txID == nil {
		op, err = b.AddCommentWithFiles(input.Message, input.Files)
		if err != nil {
			return nil, err
		}

		err = b.Commit()
		if err != nil {
			return nil, err
		}
	} else {
		op, err = bug.OpAddCommentWithFiles(input.Message)
	}

	return &models.AddCommentPayload{
		ClientMutationID: input.ClientMutationID,
		Bug:              models.NewLoadedBug(b.Snapshot()),
		Operation:        op,
	}, nil
}

func (r mutationResolver) ChangeLabels(_ context.Context, input models.ChangeLabelInput, txID *string) (*models.ChangeLabelPayload, error) {
	b, err := r.getBug(input.RepoRef, input.Prefix)
	if err != nil {
		return nil, err
	}

	results, op, err := b.ChangeLabels(input.Added, input.Removed)
	if err != nil {
		return nil, err
	}

	err = b.Commit()
	if err != nil {
		return nil, err
	}

	resultsPtr := make([]*bug.LabelChangeResult, len(results))
	for i, result := range results {
		resultsPtr[i] = &result
	}

	return &models.ChangeLabelPayload{
		ClientMutationID: input.ClientMutationID,
		Bug:              models.NewLoadedBug(b.Snapshot()),
		Operation:        op,
		Results:          resultsPtr,
	}, nil
}

func (r mutationResolver) OpenBug(_ context.Context, input models.OpenBugInput, txID *string) (*models.OpenBugPayload, error) {
	b, err := r.getBug(input.RepoRef, input.Prefix)
	if err != nil {
		return nil, err
	}

	op, err := b.Open()
	if err != nil {
		return nil, err
	}

	err = b.Commit()
	if err != nil {
		return nil, err
	}

	return &models.OpenBugPayload{
		ClientMutationID: input.ClientMutationID,
		Bug:              models.NewLoadedBug(b.Snapshot()),
		Operation:        op,
	}, nil
}

func (r mutationResolver) CloseBug(_ context.Context, input models.CloseBugInput, txID *string) (*models.CloseBugPayload, error) {
	b, err := r.getBug(input.RepoRef, input.Prefix)
	if err != nil {
		return nil, err
	}

	op, err := b.Close()
	if err != nil {
		return nil, err
	}

	err = b.Commit()
	if err != nil {
		return nil, err
	}

	return &models.CloseBugPayload{
		ClientMutationID: input.ClientMutationID,
		Bug:              models.NewLoadedBug(b.Snapshot()),
		Operation:        op,
	}, nil
}

func (r mutationResolver) SetTitle(_ context.Context, input models.SetTitleInput, txID *string) (*models.SetTitlePayload, error) {
	b, err := r.getBug(input.RepoRef, input.Prefix)
	if err != nil {
		return nil, err
	}

	op, err := b.SetTitle(input.Title)
	if err != nil {
		return nil, err
	}

	err = b.Commit()
	if err != nil {
		return nil, err
	}

	return &models.SetTitlePayload{
		ClientMutationID: input.ClientMutationID,
		Bug:              models.NewLoadedBug(b.Snapshot()),
		Operation:        op,
	}, nil
}
