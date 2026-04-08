package resolvers

import (
	"context"
	"errors"

	"github.com/git-bug/git-bug/api/graphql/connections"
	"github.com/git-bug/git-bug/api/graphql/graph"
	"github.com/git-bug/git-bug/api/graphql/models"
	"github.com/git-bug/git-bug/repository"
)

const blobTruncateSize = 1 << 20 // 1 MiB

var _ graph.GitCommitResolver = &gitCommitResolver{}

type gitCommitResolver struct{}

func (r gitCommitResolver) ShortHash(_ context.Context, obj *models.GitCommitMeta) (string, error) {
	s := string(obj.Hash)
	if len(s) > 8 {
		s = s[:8]
	}
	return s, nil
}

func (r gitCommitResolver) FullMessage(_ context.Context, obj *models.GitCommitMeta) (string, error) {
	repo := obj.Repo.BrowseRepo()
	detail, err := repo.CommitDetail(obj.Hash)
	if err != nil {
		return "", err
	}
	return detail.FullMessage, nil
}

func (r gitCommitResolver) Parents(_ context.Context, obj *models.GitCommitMeta) ([]string, error) {
	out := make([]string, len(obj.Parents))
	for i, h := range obj.Parents {
		out[i] = string(h)
	}
	return out, nil
}

func (r gitCommitResolver) Files(_ context.Context, obj *models.GitCommitMeta, after *string, before *string, first *int, last *int) (*models.GitChangedFileConnection, error) {
	repo := obj.Repo.BrowseRepo()
	detail, err := repo.CommitDetail(obj.Hash)
	if err != nil {
		return nil, err
	}

	input := models.ConnectionInput{After: after, Before: before, First: first, Last: last}
	edger := func(f repository.ChangedFile, offset int) connections.Edge {
		return connections.CursorEdge{Cursor: connections.OffsetToCursor(offset)}
	}
	conMaker := func(_ []*connections.CursorEdge, nodes []repository.ChangedFile, info *models.PageInfo, total int) (*models.GitChangedFileConnection, error) {
		ptrs := make([]*repository.ChangedFile, len(nodes))
		for i := range nodes {
			ptrs[i] = &nodes[i]
		}
		return &models.GitChangedFileConnection{Nodes: ptrs, PageInfo: info, TotalCount: total}, nil
	}
	return connections.Connection(detail.Files, edger, conMaker, input)
}

func (r gitCommitResolver) Diff(_ context.Context, obj *models.GitCommitMeta, path string) (*repository.FileDiff, error) {
	repo := obj.Repo.BrowseRepo()
	fd, err := repo.CommitFileDiff(obj.Hash, path)
	if err != nil {
		return nil, err
	}
	return &fd, nil
}

var _ graph.GitTreeEntryResolver = &gitTreeEntryResolver{}

type gitTreeEntryResolver struct{}

func (r gitTreeEntryResolver) LastCommit(_ context.Context, obj *models.GitTreeEntry) (*models.GitCommitMeta, error) {
	repo := obj.Repo.BrowseRepo()
	// Pass all sibling names so the history walk covers the whole directory,
	// which is nearly the same cost as walking for a single entry.
	// Concurrent calls for the same directory are deduplicated by a singleflight
	// inside LastCommitForEntries; subsequent calls hit the LRU cache.
	commits, err := repo.LastCommitForEntries(obj.Ref, obj.Path, obj.SiblingNames)
	if err != nil {
		return nil, err
	}
	meta, ok := commits[obj.Name]
	if !ok {
		return nil, nil
	}
	return &models.GitCommitMeta{Repo: obj.Repo, CommitMeta: meta}, nil
}

var _ graph.GitRefResolver = &gitRefResolver{}

type gitRefResolver struct{}

func (g gitRefResolver) Commit(ctx context.Context, obj *models.GitRef) (*models.GitCommitMeta, error) {
	repo := obj.Repo.BrowseRepo()
	detail, err := repo.CommitDetail(repository.Hash(obj.Hash))
	if errors.Is(err, repository.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &models.GitCommitMeta{Repo: obj.Repo, CommitMeta: detail.CommitMeta}, nil
}
