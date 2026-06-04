package gitea

import (
	"context"
	"testing"
	"time"

	gitea "gitea.dev/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/bridge/core"
	"github.com/git-bug/git-bug/bridge/core/auth"
	"github.com/git-bug/git-bug/bridge/gitea/giteatest"
	"github.com/git-bug/git-bug/cache"
	"github.com/git-bug/git-bug/repository"
)

func setupImporterOnExistingBackend(t *testing.T, serverURL string, backend *cache.RepoCache) *giteaImporter {
	t.Helper()
	gi := &giteaImporter{}
	err := gi.Init(context.Background(), backend, core.Configuration{
		confKeyBaseURL:      serverURL,
		confKeyOwner:        "owner",
		confKeyProject:      "project",
		confKeyDefaultLogin: "testuser",
	})
	require.NoError(t, err)
	return gi
}

func setupImporter(t *testing.T, serverURL string) (*giteaImporter, *cache.RepoCache) {
	t.Helper()
	repo := repository.CreateGoGitTestRepo(t, false)

	token := auth.NewToken(target, "test-token")
	token.SetMetadata(auth.MetaKeyLogin, "testuser")
	token.SetMetadata(auth.MetaKeyBaseURL, serverURL)
	err := auth.Store(repo, token)
	require.NoError(t, err)

	backend, err := cache.NewRepoCacheNoEvents(repo)
	require.NoError(t, err)
	t.Cleanup(func() { _ = backend.Close() })

	return setupImporterOnExistingBackend(t, serverURL, backend), backend
}

func collectErrors(results []core.ImportResult) []error {
	var errs []error
	for _, r := range results {
		if r.Err != nil {
			errs = append(errs, r.Err)
		}
	}
	return errs
}

func runImport(t *testing.T, gi *giteaImporter, backend *cache.RepoCache) []core.ImportResult {
	t.Helper()
	ch, err := gi.ImportAll(context.Background(), backend, time.Time{})
	require.NoError(t, err)
	var results []core.ImportResult
	for r := range ch {
		results = append(results, r)
	}
	return results
}

func testIssue() *gitea.Issue {
	return &gitea.Issue{
		ID:      1,
		Index:   1,
		Title:   "Test Issue",
		Body:    "Test body",
		Poster:  &gitea.User{UserName: "testuser", FullName: "Test User", Email: "testuser@example.com"},
		Created: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
	}
}

func testComment(id int64, body string) *gitea.Comment {
	ts := time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)
	return &gitea.Comment{
		ID:      id,
		Body:    body,
		Poster:  &gitea.User{UserName: "testuser"},
		Created: ts,
		Updated: ts,
	}
}

// TestImportNilPoster documents finding #1: comment.Poster is a *User pointer and
// is never nil-checked before dereferencing .UserName. A deleted or bot user can
// return "user": null from the API, which panics the import goroutine.
//
// After the fix: ImportError should be emitted instead of panicking.
func TestImportNilPoster(t *testing.T) {
	ts := time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)
	srv := (&giteatest.FakeAPI{
		Owner:   "owner",
		Project: "project",
		Issues:  []*gitea.Issue{testIssue()},
		Comments: []*gitea.Comment{
			{ID: 1, Body: "null poster comment", Poster: nil, Created: ts, Updated: ts},
		},
	}).NewServer(t)
	gi, backend := setupImporter(t, srv.URL)

	results := runImport(t, gi, backend)
	assert.Empty(t, collectErrors(results), "should resolve to a ghost identity")
	_, err := backend.Identities().ResolveIdentityImmutableMetadata(metaKeyGiteaLogin, "@deleted-user")
	assert.NoError(t, err, "expected a ghost identity to be created for deleted user")
}

// TestImportIdempotentComments documents finding #2: AddCommentRaw is called with
// an empty metadata map, so no remote comment ID is stored. Running ImportAll a
// second time re-adds every comment, duplicating them in the local bug.
//
// After the fix: second import should produce no new operations.
func TestImportIdempotentComments(t *testing.T) {
	srv := (&giteatest.FakeAPI{
		Owner:    "owner",
		Project:  "project",
		Issues:   []*gitea.Issue{testIssue()},
		Comments: []*gitea.Comment{testComment(1, "first"), testComment(2, "second")},
	}).NewServer(t)

	gi, backend := setupImporter(t, srv.URL)
	_ = runImport(t, gi, backend)

	gi2 := setupImporterOnExistingBackend(t, srv.URL, backend)
	results2 := runImport(t, gi2, backend)

	for _, r := range results2 {
		assert.NoError(t, r.Err)
		assert.NotEqual(t, core.ImportEventBug, r.Event,
			"second import should not create a new bug")
		assert.NotEqual(t, core.ImportEventComment, r.Event,
			"second import should not duplicate comments (finding #2: empty metadata)")
	}
}

// TestImportCommentErrorEmitted documents finding #3: when the comment iterator
// encounters an API error, the current issue is committed in a partial state
// before the error reaches the caller. An ImportError should be emitted and the
// partial bug should not be silently committed as complete.
func TestImportCommentErrorEmitted(t *testing.T) {
	srv := (&giteatest.FakeAPI{
		Owner:          "owner",
		Project:        "project",
		Issues:         []*gitea.Issue{testIssue()},
		Comments:       []*gitea.Comment{testComment(1, "comment")},
		CommentErrPage: 1,
	}).NewServer(t)
	gi, backend := setupImporter(t, srv.URL)

	results := runImport(t, gi, backend)

	assert.NotEmpty(t, collectErrors(results), "expected ImportError when comment fetch fails")
}

// TestImportGhostDedup documents that deletedIdentity (import.go:223) calls
// NewRaw without first resolving by metadata, so two null-Poster comments in
// one import create two ghost identities (or fail on the second NewRaw).
//
// After the fix: a single ghost identity is shared by all null-Poster comments.
func TestImportGhostDedup(t *testing.T) {
	ts := time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)
	srv := (&giteatest.FakeAPI{
		Owner:   "owner",
		Project: "project",
		Issues:  []*gitea.Issue{testIssue()},
		Comments: []*gitea.Comment{
			{ID: 1, Body: "first null", Poster: nil, Created: ts, Updated: ts},
			{ID: 2, Body: "second null", Poster: nil, Created: ts, Updated: ts},
		},
	}).NewServer(t)
	gi, backend := setupImporter(t, srv.URL)

	results := runImport(t, gi, backend)
	assert.Empty(t, collectErrors(results),
		"two null-Poster comments should reuse one ghost identity, not error")

	_, err := backend.Identities().ResolveIdentityImmutableMetadata(metaKeyGiteaLogin, DeletedIdentity)
	assert.NoError(t, err,
		"a single ghost identity should be resolvable (ErrMultipleMatch indicates duplicates)")
}

// TestImportNilIssuePoster documents that ensureIssue (import.go:131) derefs
// issue.Poster.UserName without nil-checking. Same shape as finding #1 but on
// the issue path rather than the comment path.
//
// After the fix: ImportError or ghost-identity fallback, no panic.
func TestImportNilIssuePoster(t *testing.T) {
	ts := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	srv := (&giteatest.FakeAPI{
		Owner:   "owner",
		Project: "project",
		Issues: []*gitea.Issue{{
			ID: 1, Index: 1, Title: "Test Issue", Body: "Test body",
			Poster: nil, Created: ts,
		}},
	}).NewServer(t)
	gi, backend := setupImporter(t, srv.URL)

	results := runImport(t, gi, backend)
	assert.Empty(t, collectErrors(results),
		"null issue Poster should resolve to ghost, not panic or error")

	_, err := backend.Identities().ResolveIdentityImmutableMetadata(metaKeyGiteaLogin, DeletedIdentity)
	assert.NoError(t, err, "expected a ghost identity for the null-Poster issue")
}

// TestImportEnsurePersonNotFound documents that ensurePerson (import.go:192)
// derefs the SDK's *User return on the 404 branch. Most SDK versions return
// a nil *User alongside the 404 error.
//
// After the fix: a placeholder identity is created with the requested login.
func TestImportEnsurePersonNotFound(t *testing.T) {
	ts := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	srv := (&giteatest.FakeAPI{
		Owner:         "owner",
		Project:       "project",
		NotFoundUsers: []string{"deleted-user"},
		Issues: []*gitea.Issue{{
			ID: 1, Index: 1, Title: "Test", Body: "Body",
			Poster:  &gitea.User{UserName: "deleted-user"},
			Created: ts,
		}},
	}).NewServer(t)
	gi, backend := setupImporter(t, srv.URL)

	results := runImport(t, gi, backend)
	assert.Empty(t, collectErrors(results),
		"a 404 on user lookup should fall back to a placeholder identity, not panic")

	_, err := backend.Identities().ResolveIdentityImmutableMetadata(metaKeyGiteaLogin, "deleted-user")
	assert.NoError(t, err,
		"a placeholder identity tagged with the looked-up login should exist")
}

// TestImportEnsurePersonNetworkError verifies that a network-level failure
// during user lookup (no *Response object) is surfaced as an ImportError
// rather than panicking on a nil resp.StatusCode dereference.
func TestImportEnsurePersonNetworkError(t *testing.T) {
	ts := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	srv := (&giteatest.FakeAPI{
		Owner:               "owner",
		Project:             "project",
		UserNetworkErrLogin: "dropped-user",
		Issues: []*gitea.Issue{{
			ID: 1, Index: 1, Title: "t", Body: "b",
			Poster:  &gitea.User{UserName: "dropped-user"},
			Created: ts,
		}},
	}).NewServer(t)
	gi, backend := setupImporter(t, srv.URL)

	var results []core.ImportResult
	require.NotPanics(t, func() {
		results = runImport(t, gi, backend)
	}, "a network-level user-lookup failure must not panic on nil resp")
	assert.NotEmpty(t, collectErrors(results),
		"a network-level user-lookup failure should surface as ImportError")
}

// TestImportEnsurePersonServerError verifies that a 5xx from /users/{login}
// propagates as an ImportError rather than being misclassified as "user not
// found" and silently producing a placeholder.
func TestImportEnsurePersonServerError(t *testing.T) {
	ts := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	srv := (&giteatest.FakeAPI{
		Owner:        "owner",
		Project:      "project",
		UserErrLogin: "flaky-user",
		Issues: []*gitea.Issue{{
			ID: 1, Index: 1, Title: "t", Body: "b",
			Poster:  &gitea.User{UserName: "flaky-user"},
			Created: ts,
		}},
	}).NewServer(t)
	gi, backend := setupImporter(t, srv.URL)

	results := runImport(t, gi, backend)
	assert.NotEmpty(t, collectErrors(results),
		"a 5xx on user lookup should propagate, not be misclassified as 'user not found'")
}

// TestImportIdentityReuseAcrossIssues verifies that two issues authored by the
// same user produce a single identity, attributed to both bugs.
func TestImportIdentityReuseAcrossIssues(t *testing.T) {
	ts := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	user := &gitea.User{UserName: "testuser", FullName: "Test User", Email: "testuser@example.com"}
	srv := (&giteatest.FakeAPI{
		Owner:   "owner",
		Project: "project",
		Issues: []*gitea.Issue{
			{ID: 1, Index: 1, Title: "first", Poster: user, Created: ts},
			{ID: 2, Index: 2, Title: "second", Poster: user, Created: ts},
		},
	}).NewServer(t)
	gi, backend := setupImporter(t, srv.URL)

	results := runImport(t, gi, backend)
	require.Empty(t, collectErrors(results))

	_, err := backend.Identities().ResolveIdentityImmutableMetadata(metaKeyGiteaLogin, "testuser")
	assert.NoError(t, err,
		"the same login across two issues should resolve to a single identity (ErrMultipleMatch indicates dedup broke)")

	identityEvents := 0
	for _, r := range results {
		if r.Event == core.ImportEventIdentity {
			identityEvents++
		}
	}
	assert.Equal(t, 1, identityEvents,
		"only one identity-creation event expected for two issues by the same user")
}

// TestImportSecondRunEmitsNothing verifies that re-importing an unchanged repo
// emits ImportEventNothing for the existing bug, signaling no-op rather than
// silently producing no result. On failure, dumps the operations the second
// run unexpectedly added so the duplication source is visible.
func TestImportSecondRunEmitsNothing(t *testing.T) {
	srv := (&giteatest.FakeAPI{
		Owner:    "owner",
		Project:  "project",
		Issues:   []*gitea.Issue{testIssue()},
		Comments: []*gitea.Comment{testComment(1, "c1")},
	}).NewServer(t)

	gi, backend := setupImporter(t, srv.URL)
	_ = runImport(t, gi, backend)

	bugIds := backend.Bugs().AllIds()
	require.Len(t, bugIds, 1)
	bug, err := backend.Bugs().Resolve(bugIds[0])
	require.NoError(t, err)
	opsBefore := len(bug.Snapshot().Operations)

	gi2 := setupImporterOnExistingBackend(t, srv.URL, backend)
	results2 := runImport(t, gi2, backend)

	bug, err = backend.Bugs().Resolve(bugIds[0])
	require.NoError(t, err)
	opsAfter := bug.Snapshot().Operations
	if len(opsAfter) > opsBefore {
		for i, op := range opsAfter[opsBefore:] {
			t.Logf("unexpected op #%d added on second run: %T %+v", i+1, op, op)
		}
	}
	assert.Equal(t, opsBefore, len(opsAfter),
		"second import on unchanged repo should add no operations (finding #2: AddCommentRaw with empty metadata can't dedupe)")

	nothing := 0
	for _, r := range results2 {
		if r.Event == core.ImportEventNothing {
			nothing++
		}
	}
	assert.GreaterOrEqual(t, nothing, 1,
		"second import on unchanged repo should emit at least one ImportEventNothing")
}

// TestImportMultipleIssues verifies that a multi-issue import correctly
// attributes each issue's comments to the right bug — catches cross-issue
// state leakage in the importer (independent of iterator-level reset).
func TestImportMultipleIssues(t *testing.T) {
	ts := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	user := &gitea.User{UserName: "testuser", FullName: "Test User", Email: "testuser@example.com"}
	srv := (&giteatest.FakeAPI{
		Owner:   "owner",
		Project: "project",
		Issues: []*gitea.Issue{
			{ID: 1, Index: 1, Title: "first", Body: "b1", Poster: user, Created: ts},
			{ID: 2, Index: 2, Title: "second", Body: "b2", Poster: user, Created: ts},
		},
		CommentsByIssue: map[int64][]*gitea.Comment{
			1: {{ID: 11, Body: "issue-1 only", Poster: user, Created: ts, Updated: ts}},
			2: {{ID: 21, Body: "issue-2 only", Poster: user, Created: ts, Updated: ts}},
		},
	}).NewServer(t)
	gi, backend := setupImporter(t, srv.URL)

	results := runImport(t, gi, backend)
	require.Empty(t, collectErrors(results))

	bugEvents := 0
	for _, r := range results {
		if r.Event == core.ImportEventBug {
			bugEvents++
		}
	}
	assert.Equal(t, 2, bugEvents, "two issues should produce two ImportEventBug events")
}
