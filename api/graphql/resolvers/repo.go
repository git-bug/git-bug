package resolvers

import (
	"bytes"
	"context"
	"io"
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

func (repoResolver) Refs(_ context.Context, obj *models.Repository, after *string, before *string, first *int, last *int, typeArg *models.GitRefType) (*models.GitRefConnection, error) {
	repo, err := obj.Repo.BrowseRepo()
	if err != nil {
		return nil, err
	}

	var refs []*models.GitRef

	if typeArg == nil || *typeArg == models.GitRefTypeBranch {
		branches, err := repo.Branches()
		if err != nil {
			return nil, err
		}
		for _, b := range branches {
			refs = append(refs, &models.GitRef{
				Name:      "refs/heads/" + b.Name,
				ShortName: b.Name,
				Type:      models.GitRefTypeBranch,
				Hash:      string(b.Hash),
				IsDefault: b.IsDefault,
			})
		}
	}

	if typeArg == nil || *typeArg == models.GitRefTypeTag {
		tags, err := repo.Tags()
		if err != nil {
			return nil, err
		}
		for _, t := range tags {
			refs = append(refs, &models.GitRef{
				Name:      "refs/tags/" + t.Name,
				ShortName: t.Name,
				Type:      models.GitRefTypeTag,
				Hash:      string(t.Hash),
			})
		}
	}

	input := models.ConnectionInput{After: after, Before: before, First: first, Last: last}
	edger := func(r *models.GitRef, offset int) connections.Edge {
		return connections.CursorEdge{Cursor: connections.OffsetToCursor(offset)}
	}
	conMaker := func(edges []*connections.CursorEdge, nodes []*models.GitRef, info *models.PageInfo, total int) (*models.GitRefConnection, error) {
		return &models.GitRefConnection{Nodes: nodes, PageInfo: info, TotalCount: total}, nil
	}
	return connections.Connection(refs, edger, conMaker, input)
}

func (repoResolver) Tree(_ context.Context, obj *models.Repository, ref string, path *string) ([]*repository.TreeEntry, error) {
	repo, err := obj.Repo.BrowseRepo()
	if err != nil {
		return nil, err
	}
	p := ""
	if path != nil {
		p = *path
	}
	entries, err := repo.TreeAtPath(ref, p)
	if err != nil {
		return nil, err
	}
	ptrs := make([]*repository.TreeEntry, len(entries))
	for i := range entries {
		ptrs[i] = &entries[i]
	}
	return ptrs, nil
}

func (repoResolver) Blob(_ context.Context, obj *models.Repository, ref string, path string) (*models.GitBlob, error) {
	repo, err := obj.Repo.BrowseRepo()
	if err != nil {
		return nil, err
	}
	rc, size, err := repo.BlobAtPath(ref, path)
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	limited := io.LimitReader(rc, blobTruncateSize+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}

	isTruncated := int64(len(data)) > blobTruncateSize
	if isTruncated {
		data = data[:blobTruncateSize]
	}

	isBinary := bytes.IndexByte(data, 0) >= 0
	blob := &models.GitBlob{
		Path:        path,
		Hash:        "", // hash not available from BlobAtPath
		Size:        int(size),
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

	repo, err := obj.Repo.BrowseRepo()
	if err != nil {
		return nil, err
	}

	p := ""
	if path != nil {
		p = *path
	}

	limit := 0
	if first != nil {
		limit = *first + 1 // fetch one extra to detect hasNextPage
	}

	var afterHash repository.Hash
	if after != nil {
		afterHash = repository.Hash(*after)
	}

	commits, err := repo.CommitLog(ref, p, limit, afterHash)
	if err != nil {
		return nil, err
	}

	// client-side since/until filtering
	if since != nil || until != nil {
		filtered := commits[:0]
		for _, c := range commits {
			if since != nil && c.Date.Before(*since) {
				continue
			}
			if until != nil && c.Date.After(*until) {
				continue
			}
			filtered = append(filtered, c)
		}
		commits = filtered
	}

	hasNextPage := false
	if first != nil && len(commits) > *first {
		hasNextPage = true
		commits = commits[:*first]
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
		TotalCount: 0, // unknown without full walk
	}, nil
}

func (repoResolver) Commit(_ context.Context, obj *models.Repository, hash string) (*models.GitCommitMeta, error) {
	repo, err := obj.Repo.BrowseRepo()
	if err != nil {
		return nil, err
	}
	commits, err := repo.CommitLog(hash, "", 1, "")
	if err != nil {
		return nil, err
	}
	if len(commits) == 0 {
		return nil, repository.ErrNotFound
	}
	return &models.GitCommitMeta{Repo: obj.Repo, CommitMeta: commits[0]}, nil
}

func (repoResolver) LastCommits(_ context.Context, obj *models.Repository, ref string, path *string, names []string) ([]*models.GitLastCommit, error) {
	repo, err := obj.Repo.BrowseRepo()
	if err != nil {
		return nil, err
	}
	p := ""
	if path != nil {
		p = *path
	}
	byName, err := repo.LastCommitForEntries(ref, p, names)
	if err != nil {
		return nil, err
	}
	result := make([]*models.GitLastCommit, 0, len(byName))
	for name, meta := range byName {
		m := meta
		result = append(result, &models.GitLastCommit{Name: name, Commit: &models.GitCommitMeta{Repo: obj.Repo, CommitMeta: m}})
	}
	return result, nil
}
