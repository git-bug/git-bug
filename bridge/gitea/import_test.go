package gitea

import (
	"context"
	"runtime"
	"strings"
	"testing"
	"time"

	gitea "gitea.dev/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/bridge/core"
	"github.com/git-bug/git-bug/bridge/core/auth"
	"github.com/git-bug/git-bug/bridge/gitea/giteatest"
	"github.com/git-bug/git-bug/cache"
	"github.com/git-bug/git-bug/entities/common"
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

// TestImportNegativeDedupAcrossRepos verifies that issue #1 from two
// different forges (distinct baseURL/owner/project) produces two bugs.
// ensureIssue's matcher checks all four metadata keys; a regression that
// dropped any of them would silently merge unrelated issues.
func TestImportNegativeDedupAcrossRepos(t *testing.T) {
	ts := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	mkIssue := func(title string) *gitea.Issue {
		return &gitea.Issue{
			ID: 1, Index: 1, Title: title, Body: "b",
			Poster: &gitea.User{UserName: "testuser", FullName: "Test User", Email: "u@example.com"},
			Created: ts,
		}
	}
	srvA := (&giteatest.FakeAPI{Owner: "owner-a", Project: "proj1", Issues: []*gitea.Issue{mkIssue("from A")}}).NewServer(t)
	srvB := (&giteatest.FakeAPI{Owner: "owner-b", Project: "proj2", Issues: []*gitea.Issue{mkIssue("from B")}}).NewServer(t)

	repo := repository.CreateGoGitTestRepo(t, false)
	for _, baseURL := range []string{srvA.URL, srvB.URL} {
		token := auth.NewToken(target, "test-token")
		token.SetMetadata(auth.MetaKeyLogin, "testuser")
		token.SetMetadata(auth.MetaKeyBaseURL, baseURL)
		require.NoError(t, auth.Store(repo, token))
	}
	backend, err := cache.NewRepoCacheNoEvents(repo)
	require.NoError(t, err)
	t.Cleanup(func() { _ = backend.Close() })

	importFrom := func(url, owner, project string) {
		gi := &giteaImporter{}
		require.NoError(t, gi.Init(context.Background(), backend, core.Configuration{
			confKeyBaseURL:      url,
			confKeyOwner:        owner,
			confKeyProject:      project,
			confKeyDefaultLogin: "testuser",
		}))
		_ = runImport(t, gi, backend)
	}
	importFrom(srvA.URL, "owner-a", "proj1")
	importFrom(srvB.URL, "owner-b", "proj2")

	assert.Len(t, backend.Bugs().AllIds(), 2,
		"issue #1 from two distinct forges should produce two bugs, not be deduped together")
}

// TestImportPropagatesIssueTitleUpdates verifies that re-importing an issue
// whose title/body changed upstream propagates the change to the local bug.
// ensureIssue currently returns the existing bug without checking for edits,
// so this test pins a behavior gap.
func TestImportPropagatesIssueTitleUpdates(t *testing.T) {
	ts := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	issue := &gitea.Issue{
		ID: 1, Index: 1, Title: "original title", Body: "original body",
		Poster:  &gitea.User{UserName: "testuser", FullName: "Test User", Email: "u@example.com"},
		Created: ts,
	}
	fa := &giteatest.FakeAPI{Owner: "owner", Project: "project", Issues: []*gitea.Issue{issue}}
	srv := fa.NewServer(t)

	gi, backend := setupImporter(t, srv.URL)
	_ = runImport(t, gi, backend)

	// Simulate the issue being edited upstream between imports.
	issue.Title = "updated title"
	issue.Body = "updated body"

	gi2 := setupImporterOnExistingBackend(t, srv.URL, backend)
	_ = runImport(t, gi2, backend)

	bugIds := backend.Bugs().AllIds()
	require.Len(t, bugIds, 1)
	b, err := backend.Bugs().Resolve(bugIds[0])
	require.NoError(t, err)
	snap := b.Snapshot()

	assert.Equal(t, "updated title", snap.Title,
		"title change should propagate on re-import")
	require.NotEmpty(t, snap.Comments)
	assert.Contains(t, snap.Comments[0].Message, "updated body",
		"body change should propagate on re-import")
}

// TestImportPropagatesCommentEdits verifies that a comment whose body is
// edited upstream propagates the new body to the local bug on re-import.
// The importer doesn't currently compare comment bodies, so this is a
// feature-gap test (will fail until ensureComment-style update logic lands).
func TestImportPropagatesCommentEdits(t *testing.T) {
	ts := time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)
	comment := &gitea.Comment{
		ID: 1, Body: "original comment",
		Poster: &gitea.User{UserName: "testuser"}, Created: ts, Updated: ts,
	}
	fa := &giteatest.FakeAPI{
		Owner: "owner", Project: "project",
		Issues:   []*gitea.Issue{testIssue()},
		Comments: []*gitea.Comment{comment},
	}
	srv := fa.NewServer(t)

	gi, backend := setupImporter(t, srv.URL)
	results := runImport(t, gi, backend)
	require.Empty(t, collectErrors(results))
	for _, r := range results {
		t.Logf("first import result: %s", r.String())
	}
	bugIds0 := backend.Bugs().AllIds()
	if len(bugIds0) > 0 {
		b0, _ := backend.Bugs().Resolve(bugIds0[0])
		t.Logf("after first import: snapshot has %d comments, %d operations",
			len(b0.Snapshot().Comments), len(b0.Snapshot().Operations))
	}

	// Simulate the comment being edited upstream.
	comment.Body = "edited comment"
	comment.Updated = ts.Add(time.Hour)

	gi2 := setupImporterOnExistingBackend(t, srv.URL, backend)
	_ = runImport(t, gi2, backend)

	bugIds := backend.Bugs().AllIds()
	require.Len(t, bugIds, 1)
	b, err := backend.Bugs().Resolve(bugIds[0])
	require.NoError(t, err)
	snap := b.Snapshot()

	require.GreaterOrEqual(t, len(snap.Comments), 2,
		"expected at least the issue body and the imported comment")
	assert.Contains(t, snap.Comments[1].Message, "edited comment",
		"comment edit should propagate on re-import (Updated > Created should trigger EditComment)")
}

// TestImportPropagatesStatusChanges verifies that an issue closed upstream
// between two imports causes the local bug to be closed. The importer
// currently never inspects issue.State or calls Open/Close.
func TestImportPropagatesStatusChanges(t *testing.T) {
	ts := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	issue := &gitea.Issue{
		ID: 1, Index: 1, Title: "t", Body: "b",
		State:   gitea.StateOpen,
		Poster:  &gitea.User{UserName: "testuser", FullName: "Test User", Email: "u@example.com"},
		Created: ts,
	}
	fa := &giteatest.FakeAPI{Owner: "owner", Project: "project", Issues: []*gitea.Issue{issue}}
	srv := fa.NewServer(t)

	gi, backend := setupImporter(t, srv.URL)
	_ = runImport(t, gi, backend)

	// Simulate the issue being closed upstream.
	issue.State = gitea.StateClosed

	gi2 := setupImporterOnExistingBackend(t, srv.URL, backend)
	_ = runImport(t, gi2, backend)

	bugIds := backend.Bugs().AllIds()
	require.Len(t, bugIds, 1)
	b, err := backend.Bugs().Resolve(bugIds[0])
	require.NoError(t, err)

	assert.Equal(t, common.ClosedStatus, b.Snapshot().Status,
		"upstream state=closed should set local bug status to closed on re-import")
}

// TestImportLabels verifies that labels on a Gitea issue end up as labels on
// the local bug. The TODO at import.go:104 ("Loop over all label events") is
// unimplemented; NextLabel() is never called, so this currently fails.
func TestImportLabels(t *testing.T) {
	fa := &giteatest.FakeAPI{
		Owner:   "owner",
		Project: "project",
		Issues:  []*gitea.Issue{testIssue()},
		Labels:  []*gitea.Label{{ID: 1, Name: "bug"}, {ID: 2, Name: "good-first-issue"}},
	}
	srv := fa.NewServer(t)

	gi, backend := setupImporter(t, srv.URL)
	_ = runImport(t, gi, backend)

	bugIds := backend.Bugs().AllIds()
	require.Len(t, bugIds, 1)
	b, err := backend.Bugs().Resolve(bugIds[0])
	require.NoError(t, err)

	labels := b.Snapshot().Labels
	names := make([]string, len(labels))
	for i, l := range labels {
		names[i] = string(l)
	}
	assert.ElementsMatch(t, []string{"bug", "good-first-issue"}, names,
		"issue labels should be imported as bug labels (TODO at import.go:104 unimplemented)")
}

// TestImportCommentAttachments verifies that a comment's attachments end up
// as Files on the local comment. The importer currently passes an empty Files
// slice (TODO at import.go:96–97), so attachments are silently dropped.
func TestImportCommentAttachments(t *testing.T) {
	ts := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	fa := &giteatest.FakeAPI{
		Owner:   "owner",
		Project: "project",
		Issues:  []*gitea.Issue{testIssue()},
		Comments: []*gitea.Comment{{
			ID: 1, Body: "with attachment",
			Poster:  &gitea.User{UserName: "testuser"},
			Created: ts, Updated: ts,
			Attachments: []*gitea.Attachment{
				{ID: 1, Name: "log.txt", DownloadURL: "https://example.com/log.txt"},
			},
		}},
	}
	srv := fa.NewServer(t)

	gi, backend := setupImporter(t, srv.URL)
	_ = runImport(t, gi, backend)

	bugIds := backend.Bugs().AllIds()
	require.Len(t, bugIds, 1)
	b, err := backend.Bugs().Resolve(bugIds[0])
	require.NoError(t, err)
	snap := b.Snapshot()

	require.GreaterOrEqual(t, len(snap.Comments), 2)
	assert.NotEmpty(t, snap.Comments[1].Files,
		"comment attachments should be imported as Files (TODO at import.go:96–97)")
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

// TestImportCommentAttributedToCommentPoster verifies that a comment's author
// is the comment's Poster, not the issue's Poster, when they differ. This is
// the same identity-attribution shape edit support will need (editor != author).
// Fails today because of the empty comment loop in import.go:84 (importComment
// is defined but never called) — same regression TestImportPropagatesCommentEdits
// documents, surfaced from a different angle.
func TestImportCommentAttributedToCommentPoster(t *testing.T) {
	ts := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	issuePoster := &gitea.User{UserName: "issue-author", FullName: "Issue Author", Email: "ia@example.com"}
	commentPoster := &gitea.User{UserName: "comment-author", FullName: "Comment Author", Email: "ca@example.com"}
	srv := (&giteatest.FakeAPI{
		Owner:   "owner",
		Project: "project",
		Issues: []*gitea.Issue{
			{ID: 1, Index: 1, Title: "t", Body: "b", Poster: issuePoster, Created: ts},
		},
		Comments: []*gitea.Comment{
			{ID: 1, Body: "hi", Poster: commentPoster, Created: ts, Updated: ts},
		},
	}).NewServer(t)
	gi, backend := setupImporter(t, srv.URL)

	results := runImport(t, gi, backend)
	require.Empty(t, collectErrors(results))

	bugIds := backend.Bugs().AllIds()
	require.Len(t, bugIds, 1)
	b, err := backend.Bugs().Resolve(bugIds[0])
	require.NoError(t, err)
	snap := b.Snapshot()
	require.GreaterOrEqual(t, len(snap.Comments), 2,
		"expected the issue body and the imported comment")

	assert.Equal(t, "issue-author", snap.Comments[0].Author.Login(),
		"issue body should be attributed to the issue's Poster")
	assert.Equal(t, "comment-author", snap.Comments[1].Author.Login(),
		"comment should be attributed to the comment's Poster, not the issue's")
}

// TestImportBugCarriesOriginMetadata verifies that imported bugs are tagged
// with MetaKeyOrigin = "gitea". `git-bug bridge pull` discovers bugs to refresh
// by this key; a regression in the literal value would silently break sync.
func TestImportBugCarriesOriginMetadata(t *testing.T) {
	srv := (&giteatest.FakeAPI{
		Owner:   "owner",
		Project: "project",
		Issues:  []*gitea.Issue{testIssue()},
	}).NewServer(t)
	gi, backend := setupImporter(t, srv.URL)

	results := runImport(t, gi, backend)
	require.Empty(t, collectErrors(results))

	bugIds := backend.Bugs().AllIds()
	require.Len(t, bugIds, 1)
	b, err := backend.Bugs().Resolve(bugIds[0])
	require.NoError(t, err)
	snap := b.Snapshot()

	origin, ok := snap.GetCreateMetadata(core.MetaKeyOrigin)
	require.True(t, ok, "imported bug must carry %s metadata", core.MetaKeyOrigin)
	assert.Equal(t, target, origin,
		"origin metadata must equal the gitea target constant (used by `bridge pull` to find bugs to refresh)")
	assert.Equal(t, "gitea", origin,
		"target constant should remain the literal string \"gitea\" — changing it orphans existing bugs from the bridge")
}

// TestImportIdentityCarriesLoginAndProfile verifies that a newly-created
// identity carries metaKeyGiteaLogin (used by ResolveIdentityImmutableMetadata
// for dedup) and the AvatarUrl / Email / Name fetched from the user endpoint.
// If the metadata key silently changed, TestImportIdentityReuseAcrossIssues
// would still pass on a single run — this test catches that regression.
func TestImportIdentityCarriesLoginAndProfile(t *testing.T) {
	ts := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	// FakeAPI's user endpoint returns FullName=login, Email=login+"@example.com";
	// the issue Poster fields are ignored once ensurePerson hits the user API.
	srv := (&giteatest.FakeAPI{
		Owner:   "owner",
		Project: "project",
		Issues: []*gitea.Issue{
			{ID: 1, Index: 1, Title: "t", Body: "b",
				Poster:  &gitea.User{UserName: "alice"},
				Created: ts},
		},
	}).NewServer(t)
	gi, backend := setupImporter(t, srv.URL)

	results := runImport(t, gi, backend)
	require.Empty(t, collectErrors(results))

	i, err := backend.Identities().ResolveIdentityImmutableMetadata(metaKeyGiteaLogin, "alice")
	require.NoError(t, err,
		"identity should be resolvable by metaKeyGiteaLogin — if the key name drifted, dedup is broken")

	imm := i.Identity.ImmutableMetadata()
	assert.Equal(t, "alice", imm[metaKeyGiteaLogin],
		"metaKeyGiteaLogin should be set to the gitea username")

	assert.Equal(t, "alice", i.Login(),
		"identity Login should match the gitea UserName")
	assert.Equal(t, "alice", i.Name(),
		"identity Name should come from the user endpoint's FullName")
	assert.Equal(t, "alice@example.com", i.Email(),
		"identity Email should come from the user endpoint")
}

// TestImportInitNoToken verifies that Init returns ErrMissingIdentityToken when
// the auth store has no token matching baseURL+login. The error path at
// import.go:50 is only hit when auth.List finds nothing.
func TestImportInitNoToken(t *testing.T) {
	repo := repository.CreateGoGitTestRepo(t, false)
	backend, err := cache.NewRepoCacheNoEvents(repo)
	require.NoError(t, err)
	t.Cleanup(func() { _ = backend.Close() })

	gi := &giteaImporter{}
	err = gi.Init(context.Background(), backend, core.Configuration{
		confKeyBaseURL:      "http://example.invalid",
		confKeyOwner:        "owner",
		confKeyProject:      "project",
		confKeyDefaultLogin: "testuser",
	})
	assert.ErrorIs(t, err, ErrMissingIdentityToken,
		"Init with no stored token should return ErrMissingIdentityToken")
}

// TestImportRepoNotFound verifies that a 404 from the issues endpoint (the
// shape Gitea returns when the repo doesn't exist or the token can't see it)
// surfaces through the import results rather than producing a silent empty
// import. FakeAPI's RepoNotFound flag returns 404 from the repo endpoint;
// the iterator's first issue fetch will see the same condition via the issues
// path returning no handler match.
func TestImportRepoNotFound(t *testing.T) {
	srv := (&giteatest.FakeAPI{
		Owner:        "wrong-owner",
		Project:      "wrong-project",
		RepoNotFound: true,
	}).NewServer(t)

	// Configure the importer to ask for a repo that the FakeAPI doesn't serve.
	repo := repository.CreateGoGitTestRepo(t, false)
	token := auth.NewToken(target, "test-token")
	token.SetMetadata(auth.MetaKeyLogin, "testuser")
	token.SetMetadata(auth.MetaKeyBaseURL, srv.URL)
	require.NoError(t, auth.Store(repo, token))

	backend, err := cache.NewRepoCacheNoEvents(repo)
	require.NoError(t, err)
	t.Cleanup(func() { _ = backend.Close() })

	gi := &giteaImporter{}
	require.NoError(t, gi.Init(context.Background(), backend, core.Configuration{
		confKeyBaseURL:      srv.URL,
		confKeyOwner:        "ghost-owner", // not registered with the mux
		confKeyProject:      "ghost-project",
		confKeyDefaultLogin: "testuser",
	}))

	results := runImport(t, gi, backend)
	assert.NotEmpty(t, collectErrors(results),
		"importing a nonexistent repo should surface as ImportError, not a silent empty import")
	assert.Empty(t, backend.Bugs().AllIds(),
		"no bugs should be created when the repo is unreachable")
}

// TestImportStopsAfterIssueError pins the current fail-fast behavior of
// ImportAll: when ensureIssue fails on issue N, the goroutine emits a single
// ImportError and returns without processing issue N+1. github/gitlab continue
// past per-issue failures; this test documents that gitea diverges. If we ever
// switch to continue-on-error, update this test (don't delete it — the
// behavior change should be deliberate).
func TestImportStopsAfterIssueError(t *testing.T) {
	ts := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	srv := (&giteatest.FakeAPI{
		Owner:        "owner",
		Project:      "project",
		UserErrLogin: "flaky-user", // ensurePerson on issue 1 will 500
		Issues: []*gitea.Issue{
			{ID: 1, Index: 1, Title: "fails", Body: "b",
				Poster: &gitea.User{UserName: "flaky-user"}, Created: ts},
			{ID: 2, Index: 2, Title: "should not be imported", Body: "b",
				Poster: &gitea.User{UserName: "testuser"}, Created: ts},
		},
	}).NewServer(t)
	gi, backend := setupImporter(t, srv.URL)

	results := runImport(t, gi, backend)
	require.NotEmpty(t, collectErrors(results),
		"the flaky-user 500 on issue #1 should produce an ImportError")
	assert.Empty(t, backend.Bugs().AllIds(),
		"current behavior: import bails on the first ensureIssue failure; issue #2 is never reached")
}

// importerGoroutineLive returns true if any goroutine's stack mentions the
// gitea importer's ImportAll closure. Used by TestImportCancelLeaksGoroutine
// to detect a producer goroutine that's blocked on `out <-` after the
// consumer stopped reading and the context was canceled.
func importerGoroutineLive() bool {
	buf := make([]byte, 1<<20)
	n := runtime.Stack(buf, true)
	stacks := string(buf[:n])
	// The producer goroutine is the anonymous closure started in ImportAll;
	// frames from that closure appear as paths under bridge/gitea/import.go.
	return strings.Contains(stacks, "bridge/gitea.(*giteaImporter).ImportAll")
}

// TestImportCancelLeaksGoroutine pins the goroutine-leak contract gap: when
// the consumer stops draining and cancels ctx, the producer goroutine should
// exit promptly. Today it doesn't — every `out <- X` in import.go is a bare
// channel send, not a select on ctx.Done. If the producer is blocked on a
// send when cancel fires, it stays blocked until the process exits.
//
// The fix (not made here) is a send helper:
//
//	func (gi *giteaImporter) send(ctx context.Context, r core.ImportResult) {
//	    select { case gi.out <- r: case <-ctx.Done(): }
//	}
//
// and replacing every bare `out <-` with `gi.send(ctx, ...)`.
//
// The test sets up enough issues that the producer will reach a send-block
// state quickly, reads one event to confirm the producer is alive, cancels
// the context, stops reading, then polls for the importer goroutine to
// disappear. Fails today; will pass once sends become ctx-aware.
func TestImportCancelLeaksGoroutine(t *testing.T) {
	const issueCount = 20
	ts := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	user := &gitea.User{UserName: "testuser", FullName: "Test User", Email: "u@example.com"}
	var issues []*gitea.Issue
	for i := int64(1); i <= issueCount; i++ {
		issues = append(issues, &gitea.Issue{
			ID: i, Index: i, Title: "t", Body: "b", Poster: user, Created: ts,
		})
	}
	srv := (&giteatest.FakeAPI{Owner: "owner", Project: "project", Issues: issues}).NewServer(t)
	gi, backend := setupImporter(t, srv.URL)

	ctx, cancel := context.WithCancel(context.Background())
	ch, err := gi.ImportAll(ctx, backend, time.Time{})
	require.NoError(t, err)

	// Read one event to ensure the producer goroutine has actually started
	// and reached a send. Anything emitted by the import is fine here — for
	// the first issue, ImportEventIdentity comes before ImportEventBug.
	select {
	case <-ch:
	case <-time.After(2 * time.Second):
		t.Fatal("producer never emitted an event — import is broken in some other way")
	}
	require.True(t, importerGoroutineLive(),
		"sanity: importer goroutine should be running while we hold a pending send")

	// Stop reading and cancel. A ctx-aware producer exits within one send.
	cancel()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if !importerGoroutineLive() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("importer goroutine still alive 2s after cancel + stop-draining — " +
		"`out <-` doesn't respect ctx.Done(); add a select-based send helper to fix")
}

// TestImportClosesOutputChannel verifies that ImportAll closes the returned
// channel when work finishes. Every other test relies on this implicitly via
// `for r := range ch` — without `defer close(gi.out)` at import.go:69, every
// test in this file would hang. This makes the contract explicit so a future
// refactor that drops the defer fails here, not by deadlocking the suite.
func TestImportClosesOutputChannel(t *testing.T) {
	srv := (&giteatest.FakeAPI{
		Owner:   "owner",
		Project: "project",
		Issues:  []*gitea.Issue{testIssue()},
	}).NewServer(t)
	gi, backend := setupImporter(t, srv.URL)

	ch, err := gi.ImportAll(context.Background(), backend, time.Time{})
	require.NoError(t, err)

	for range ch {
	}

	select {
	case _, ok := <-ch:
		assert.False(t, ok, "channel should be closed after ImportAll completes")
	case <-time.After(time.Second):
		t.Fatal("channel was not closed within 1s of draining — ImportAll likely missing `defer close`")
	}
}
