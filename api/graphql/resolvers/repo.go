package resolvers

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"sort"
	"time"

	"github.com/git-bug/git-bug/api/auth"
	"github.com/git-bug/git-bug/api/graphql/connections"
	"github.com/git-bug/git-bug/api/graphql/graph"
	"github.com/git-bug/git-bug/api/graphql/models"
	"github.com/git-bug/git-bug/entities/common"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/query"
	"github.com/git-bug/git-bug/repository"
)

var _ graph.RepositoryResolver = &repoResolver{}

type repoResolver struct{}

func (repoResolver) Name(_ context.Context, obj *models.Repository) (*string, error) {
	if obj.Repo.IsDefaultRepo() {
		return nil, nil
	}
	name := obj.Repo.Name()
	return &name, nil
}

func (repoResolver) AllBugs(_ context.Context, obj *models.Repository, after *string, before *string, first *int, last *int, queryStr *string) (*models.BugConnection, error) {
	input := models.ConnectionInput{
		Before: before,
		After:  after,
		First:  first,
		Last:   last,
	}

	var q *query.Query
	if queryStr != nil {
		query2, err := query.Parse(*queryStr)
		if err != nil {
			return nil, err
		}
		q = query2
	} else {
		q = query.NewQuery()
	}

	// Simply pass a []string with the ids to the pagination algorithm
	source, err := obj.Repo.Bugs().Query(q)
	if err != nil {
		return nil, err
	}

	// The edger create a custom edge holding just the id
	edger := func(id entity.Id, offset int) connections.Edge {
		return connections.LazyBugEdge{
			Id:     id,
			Cursor: connections.OffsetToCursor(offset),
		}
	}

	// The conMaker will finally load and compile bugs from git to replace the selected edges
	conMaker := func(lazyBugEdges []*connections.LazyBugEdge, lazyNode []entity.Id, info *models.PageInfo, totalCount int) (*models.BugConnection, error) {
		edges := make([]*models.BugEdge, len(lazyBugEdges))
		nodes := make([]models.BugWrapper, len(lazyBugEdges))

		for i, lazyBugEdge := range lazyBugEdges {
			excerpt, err := obj.Repo.Bugs().ResolveExcerpt(lazyBugEdge.Id)
			if err != nil {
				return nil, err
			}

			b := models.NewLazyBug(obj.Repo, excerpt)

			edges[i] = &models.BugEdge{
				Cursor: lazyBugEdge.Cursor,
				Node:   b,
			}
			nodes[i] = b
		}

		return &models.BugConnection{
			Edges:      edges,
			Nodes:      nodes,
			PageInfo:   info,
			TotalCount: totalCount,
		}, nil
	}

	return connections.Connection(source, edger, conMaker, input)
}

func (repoResolver) Bug(_ context.Context, obj *models.Repository, prefix string) (models.BugWrapper, error) {
	excerpt, err := obj.Repo.Bugs().ResolveExcerptPrefix(prefix)
	if entity.IsErrNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return models.NewLazyBug(obj.Repo, excerpt), nil
}

func (repoResolver) AllIdentities(_ context.Context, obj *models.Repository, after *string, before *string, first *int, last *int) (*models.IdentityConnection, error) {
	input := models.ConnectionInput{
		Before: before,
		After:  after,
		First:  first,
		Last:   last,
	}

	// Simply pass a []string with the ids to the pagination algorithm
	source := obj.Repo.Identities().AllIds()

	// The edger create a custom edge holding just the id
	edger := func(id entity.Id, offset int) connections.Edge {
		return connections.LazyIdentityEdge{
			Id:     id,
			Cursor: connections.OffsetToCursor(offset),
		}
	}

	// The conMaker will finally load and compile identities from git to replace the selected edges
	conMaker := func(lazyIdentityEdges []*connections.LazyIdentityEdge, lazyNode []entity.Id, info *models.PageInfo, totalCount int) (*models.IdentityConnection, error) {
		edges := make([]*models.IdentityEdge, len(lazyIdentityEdges))
		nodes := make([]models.IdentityWrapper, len(lazyIdentityEdges))

		for k, lazyIdentityEdge := range lazyIdentityEdges {
			excerpt, err := obj.Repo.Identities().ResolveExcerpt(lazyIdentityEdge.Id)
			if err != nil {
				return nil, err
			}

			i := models.NewLazyIdentity(obj.Repo, excerpt)

			edges[k] = &models.IdentityEdge{
				Cursor: lazyIdentityEdge.Cursor,
				Node:   i,
			}
			nodes[k] = i
		}

		return &models.IdentityConnection{
			Edges:      edges,
			Nodes:      nodes,
			PageInfo:   info,
			TotalCount: totalCount,
		}, nil
	}

	return connections.Connection(source, edger, conMaker, input)
}

func (repoResolver) Identity(_ context.Context, obj *models.Repository, prefix string) (models.IdentityWrapper, error) {
	excerpt, err := obj.Repo.Identities().ResolveExcerptPrefix(prefix)
	if entity.IsErrNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return models.NewLazyIdentity(obj.Repo, excerpt), nil
}

func (repoResolver) UserIdentity(ctx context.Context, obj *models.Repository) (models.IdentityWrapper, error) {
	id, err := auth.UserFromCtx(ctx, obj.Repo)
	if err == auth.ErrNotAuthenticated {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return models.NewLoadedIdentity(id.Identity), nil
}

func (repoResolver) ValidLabels(_ context.Context, obj *models.Repository, after *string, before *string, first *int, last *int) (*models.LabelConnection, error) {
	input := models.ConnectionInput{
		Before: before,
		After:  after,
		First:  first,
		Last:   last,
	}

	edger := func(label common.Label, offset int) connections.Edge {
		return models.LabelEdge{
			Node:   label,
			Cursor: connections.OffsetToCursor(offset),
		}
	}

	conMaker := func(edges []*models.LabelEdge, nodes []common.Label, info *models.PageInfo, totalCount int) (*models.LabelConnection, error) {
		return &models.LabelConnection{
			Edges:      edges,
			Nodes:      nodes,
			PageInfo:   info,
			TotalCount: totalCount,
		}, nil
	}

	return connections.Connection(obj.Repo.Bugs().ValidLabels(), edger, conMaker, input)
}

func (repoResolver) Refs(_ context.Context, obj *models.Repository, after *string, before *string, first *int, last *int, typeArg *repository.GitRefType) (*models.GitRefConnection, error) {
	repo := obj.Repo.BrowseRepo()

	var refs []*models.GitRef

	if typeArg != nil && *typeArg == repository.GitRefTypeCommit {
		return nil, fmt.Errorf("refs: COMMIT is not a valid filter; use BRANCH or TAG")
	}

	if typeArg == nil || *typeArg == repository.GitRefTypeBranch {
		branches, err := repo.Branches()
		if err != nil {
			return nil, err
		}
		for _, b := range branches {
			refs = append(refs, &models.GitRef{
				Repo: obj.Repo,
				RefMeta: repository.RefMeta{
					Name:      "refs/heads/" + b.Name,
					ShortName: b.Name,
					Type:      repository.GitRefTypeBranch,
					Hash:      string(b.Hash),
				},
			})
		}
	}

	if typeArg == nil || *typeArg == repository.GitRefTypeTag {
		tags, err := repo.Tags()
		if err != nil {
			return nil, err
		}
		for _, t := range tags {
			refs = append(refs, &models.GitRef{
				Repo: obj.Repo,
				RefMeta: repository.RefMeta{
					Name:      "refs/tags/" + t.Name,
					ShortName: t.Name,
					Type:      repository.GitRefTypeTag,
					Hash:      string(t.Hash),
				},
			})
		}
	}

	// Sort by type (branches before tags) then by short name for stable cursors.
	sort.Slice(refs, func(i, j int) bool {
		if refs[i].Type != refs[j].Type {
			return refs[i].Type < refs[j].Type
		}
		return refs[i].ShortName < refs[j].ShortName
	})

	input := models.ConnectionInput{After: after, Before: before, First: first, Last: last}
	edger := func(r *models.GitRef, offset int) connections.Edge {
		return connections.CursorEdge{Cursor: connections.OffsetToCursor(offset)}
	}
	conMaker := func(edges []*connections.CursorEdge, nodes []*models.GitRef, info *models.PageInfo, total int) (*models.GitRefConnection, error) {
		return &models.GitRefConnection{Nodes: nodes, PageInfo: info, TotalCount: total}, nil
	}
	return connections.Connection(refs, edger, conMaker, input)
}

func (repoResolver) Tree(_ context.Context, obj *models.Repository, ref string, path *string) ([]*models.GitTreeEntry, error) {
	repo := obj.Repo.BrowseRepo()
	p := ""
	if path != nil {
		p = *path
	}
	entries, err := repo.TreeAtPath(ref, p)
	if err != nil {
		return nil, err
	}
	names := make([]string, len(entries))
	for i, e := range entries {
		names[i] = e.Name
	}
	ptrs := make([]*models.GitTreeEntry, len(entries))
	for i := range entries {
		ptrs[i] = &models.GitTreeEntry{Repo: obj.Repo, Ref: ref, Path: p, SiblingNames: names, TreeEntry: entries[i]}
	}
	return ptrs, nil
}

func (repoResolver) Blob(_ context.Context, obj *models.Repository, ref string, path string) (*models.GitBlob, error) {
	repo := obj.Repo.BrowseRepo()
	rc, size, hash, err := repo.BlobAtPath(ref, path)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	limited := io.LimitReader(rc, blobTruncateSize+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}

	// Binary detection: same heuristic as git — a null byte anywhere in the
	// content means binary. Git caps its probe at 8000 bytes; we probe all
	// bytes read (up to blobTruncateSize+1) before slicing, so a NUL in the
	// extra byte also triggers the flag. Files whose first blobTruncateSize
	// bytes are all non-NUL will be reported as text even if the remainder is
	// binary; this is a documented prefix-based heuristic.
	isBinary := bytes.IndexByte(data, 0) >= 0

	isTruncated := int64(len(data)) > blobTruncateSize
	if isTruncated {
		data = data[:blobTruncateSize]
	}

	blob := &models.GitBlob{
		Path: path,
		Hash: string(hash),
		// GraphQL Int is 32-bit; clamp to avoid overflow on 32-bit platforms or for
		// exceptionally large files (which will be truncated anyway).
		Size:        int(min(size, int64(math.MaxInt32))),
		IsBinary:    isBinary,
		IsTruncated: isTruncated,
	}
	if !isBinary {
		text := string(data)
		blob.Text = &text
	}
	return blob, nil
}

func (repoResolver) Commits(_ context.Context, obj *models.Repository, after *string, first *int, ref string, path *string, since *time.Time, until *time.Time) (*models.GitCommitConnection, error) {
	// This is not using the normal relay pagination (connection.Connection()), because that requires having the
	// full list in memory. Here, go-git does a partial walk only, which is better.

	repo := obj.Repo.BrowseRepo()

	p := ""
	if path != nil {
		p = *path
	}

	const defaultFirst = 20
	const maxFirst = 100

	n := defaultFirst
	if first != nil {
		n = *first
		if n > maxFirst {
			n = maxFirst
		}
	}
	limit := n + 1 // fetch one extra to detect hasNextPage

	var afterHash repository.Hash
	if after != nil {
		afterHash = repository.Hash(*after)
	}

	commits, err := repo.CommitLog(ref, p, limit, afterHash, since, until)
	if err != nil {
		return nil, err
	}

	hasNextPage := false
	if len(commits) > n {
		hasNextPage = true
		commits = commits[:n]
	}

	nodes := make([]*models.GitCommitMeta, len(commits))
	for i := range commits {
		nodes[i] = &models.GitCommitMeta{Repo: obj.Repo, CommitMeta: commits[i]}
	}

	startCursor := ""
	endCursor := ""
	if len(nodes) > 0 {
		startCursor = string(nodes[0].Hash)
		endCursor = string(nodes[len(nodes)-1].Hash)
	}

	return &models.GitCommitConnection{
		Nodes: nodes,
		PageInfo: &models.PageInfo{
			HasNextPage:     hasNextPage,
			HasPreviousPage: after != nil,
			StartCursor:     startCursor,
			EndCursor:       endCursor,
		},
		TotalCount: len(nodes), // lower bound; exact total unknown without full walk
	}, nil
}

func (repoResolver) Commit(_ context.Context, obj *models.Repository, hash string) (*models.GitCommitMeta, error) {
	repo := obj.Repo.BrowseRepo()
	detail, err := repo.CommitDetail(repository.Hash(hash))
	if errors.Is(err, repository.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &models.GitCommitMeta{Repo: obj.Repo, CommitMeta: detail.CommitMeta}, nil
}

func (repoResolver) LastCommits(_ context.Context, obj *models.Repository, ref string, path *string, names []string) ([]*models.GitLastCommit, error) {
	repo := obj.Repo.BrowseRepo()
	p := ""
	if path != nil {
		p = *path
	}
	byName, err := repo.LastCommitForEntries(ref, p, names)
	if err != nil {
		return nil, err
	}
	// Iterate over the input names to preserve caller-specified order.
	result := make([]*models.GitLastCommit, 0, len(names))
	for _, name := range names {
		if meta, ok := byName[name]; ok {
			m := meta
			result = append(result, &models.GitLastCommit{Name: name, Commit: &models.GitCommitMeta{Repo: obj.Repo, CommitMeta: m}})
		}
	}
	return result, nil
}

func (repoResolver) Head(_ context.Context, obj *models.Repository) (*models.GitRef, error) {
	meta, err := obj.Repo.BrowseRepo().Head()
	if errors.Is(err, repository.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &models.GitRef{Repo: obj.Repo, RefMeta: meta}, nil
}
