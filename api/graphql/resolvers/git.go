package resolvers

import (
	"context"

	"github.com/git-bug/git-bug/api/graphql/connections"
	"github.com/git-bug/git-bug/api/graphql/graph"
	"github.com/git-bug/git-bug/api/graphql/models"
	"github.com/git-bug/git-bug/cache"
	"github.com/git-bug/git-bug/repository"
)

const blobTruncateSize = 1 << 20 // 1 MiB

var _ graph.GitCommitResolver = &gitCommitResolver{}

type gitCommitResolver struct {
	cache *cache.MultiRepoCache
}

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
