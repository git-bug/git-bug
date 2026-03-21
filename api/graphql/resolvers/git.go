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

// ── gitCommitResolver ────────────────────────────────────────────────────────

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
	repo, err := obj.Repo.BrowseRepo()
	if err != nil {
		return "", err
	}
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
	repo, err := obj.Repo.BrowseRepo()
	if err != nil {
		return nil, err
	}
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
	repo, err := obj.Repo.BrowseRepo()
	if err != nil {
		return nil, err
	}
	fd, err := repo.CommitFileDiff(obj.Hash, path)
	if err != nil {
		return nil, err
	}
	return &fd, nil
}

// ── gitChangedFileResolver ───────────────────────────────────────────────────

var _ graph.GitChangedFileResolver = &gitChangedFileResolver{}

type gitChangedFileResolver struct{}

func (gitChangedFileResolver) Status(_ context.Context, obj *repository.ChangedFile) (models.GitChangeStatus, error) {
	switch obj.Status {
	case repository.ChangeStatusAdded:
		return models.GitChangeStatusAdded, nil
	case repository.ChangeStatusModified:
		return models.GitChangeStatusModified, nil
	case repository.ChangeStatusDeleted:
		return models.GitChangeStatusDeleted, nil
	case repository.ChangeStatusRenamed:
		return models.GitChangeStatusRenamed, nil
	default:
		return models.GitChangeStatusModified, nil
	}
}

// ── gitDiffLineResolver ──────────────────────────────────────────────────────

var _ graph.GitDiffLineResolver = &gitDiffLineResolver{}

type gitDiffLineResolver struct{}

func (gitDiffLineResolver) Type(_ context.Context, obj *repository.DiffLine) (models.GitDiffLineType, error) {
	switch obj.Type {
	case repository.DiffLineAdded:
		return models.GitDiffLineTypeAdded, nil
	case repository.DiffLineDeleted:
		return models.GitDiffLineTypeDeleted, nil
	default:
		return models.GitDiffLineTypeContext, nil
	}
}

// ── gitTreeEntryResolver ─────────────────────────────────────────────────────

var _ graph.GitTreeEntryResolver = &gitTreeEntryResolver{}

type gitTreeEntryResolver struct{}

func (gitTreeEntryResolver) Type(_ context.Context, obj *repository.TreeEntry) (models.GitObjectType, error) {
	switch obj.ObjectType {
	case repository.Tree:
		return models.GitObjectTypeTree, nil
	case repository.Symlink:
		return models.GitObjectTypeSymlink, nil
	case repository.Submodule:
		return models.GitObjectTypeSubmodule, nil
	default:
		return models.GitObjectTypeBlob, nil
	}
}
