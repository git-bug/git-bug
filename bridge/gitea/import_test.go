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
	bugpkg "github.com/git-bug/git-bug/entities/bug"
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

func runImportSince(t *testing.T, gi *giteaImporter, backend *cache.RepoCache, since time.Time) []core.ImportResult {
	t.Helper()
	ch, err := gi.ImportAll(context.Background(), backend, since)
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

func onlyBug(t *testing.T, backend *cache.RepoCache) *cache.BugCache {
	t.Helper()
	bugIds := backend.Bugs().AllIds()
	require.Len(t, bugIds, 1)
	b, err := backend.Bugs().Resolve(bugIds[0])
	require.NoError(t, err)
	return b
}

func onlyImportedGiteaBug(t *testing.T, backend *cache.RepoCache) *cache.BugCache {
	t.Helper()
	var matches []*cache.BugCache
	for _, id := range backend.Bugs().AllIds() {
		b, err := backend.Bugs().Resolve(id)
		require.NoError(t, err)
		origin, ok := b.Snapshot().GetCreateMetadata(core.MetaKeyOrigin)
		if ok && origin == target {
			matches = append(matches, b)
		}
	}
	require.Len(t, matches, 1)
	return matches[0]
}

func countStatusOps(b *cache.BugCache) int {
	count := 0
	for _, op := range b.Snapshot().Operations {
		if _, ok := op.(*bugpkg.SetStatusOperation); ok {
			count++
		}
	}
	return count
}

func countTitleOps(b *cache.BugCache) int {
	count := 0
	for _, op := range b.Snapshot().Operations {
		if _, ok := op.(*bugpkg.SetTitleOperation); ok {
			count++
		}
	}
	return count
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

func TestImportTitleZeroWidthSpace(t *testing.T) {
	issue := testIssue()
	issue.Title = "\u200b"
	srv := (&giteatest.FakeAPI{
		Owner: "owner", Project: "project",
		Issues: []*gitea.Issue{issue},
	}).NewServer(t)
	gi, backend := setupImporter(t, srv.URL)

	results := runImport(t, gi, backend)
	assert.Empty(t, collectErrors(results),
		"zero-width-only titles should fall back to a non-empty placeholder, not fail import")

	b := onlyBug(t, backend)
	assert.NotEmpty(t, b.Snapshot().Title)
}

func TestImportTitleAllWhitespace(t *testing.T) {
	issue := testIssue()
	issue.Title = "   "
	srv := (&giteatest.FakeAPI{
		Owner: "owner", Project: "project",
		Issues: []*gitea.Issue{issue},
	}).NewServer(t)
	gi, backend := setupImporter(t, srv.URL)

	results := runImport(t, gi, backend)
	assert.Empty(t, collectErrors(results),
		"all-whitespace titles should fall back to a non-empty placeholder, not fail import")

	b := onlyBug(t, backend)
	assert.NotEmpty(t, b.Snapshot().Title)
}

func TestImportTitleEmbeddedControlChars(t *testing.T) {
	issue := testIssue()
	issue.Title = "hello\x00world"
	srv := (&giteatest.FakeAPI{
		Owner: "owner", Project: "project",
		Issues: []*gitea.Issue{issue},
	}).NewServer(t)
	gi, backend := setupImporter(t, srv.URL)

	results := runImport(t, gi, backend)
	require.Empty(t, collectErrors(results))

	b := onlyBug(t, backend)
	assert.Equal(t, "helloworld", b.Snapshot().Title)
}

func TestImportStatusChangeIdempotent(t *testing.T) {
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

	issue.State = gitea.StateClosed
	_ = runImport(t, setupImporterOnExistingBackend(t, srv.URL, backend), backend)
	_ = runImport(t, setupImporterOnExistingBackend(t, srv.URL, backend), backend)

	b := onlyBug(t, backend)
	assert.Equal(t, common.ClosedStatus, b.Snapshot().Status)
	assert.Equal(t, 1, countStatusOps(b),
		"re-importing the same upstream state change should not duplicate SetStatus operations")
}

func TestImportTitleChangeIdempotent(t *testing.T) {
	issue := testIssue()
	issue.Title = "original"
	fa := &giteatest.FakeAPI{Owner: "owner", Project: "project", Issues: []*gitea.Issue{issue}}
	srv := fa.NewServer(t)
	gi, backend := setupImporter(t, srv.URL)
	_ = runImport(t, gi, backend)

	issue.Title = "updated"
	_ = runImport(t, setupImporterOnExistingBackend(t, srv.URL, backend), backend)
	_ = runImport(t, setupImporterOnExistingBackend(t, srv.URL, backend), backend)

	b := onlyBug(t, backend)
	assert.Equal(t, "updated", b.Snapshot().Title)
	assert.Equal(t, 1, countTitleOps(b),
		"re-importing the same upstream title change should not duplicate SetTitle operations")
}

func TestImportBugMetadataKeysExact(t *testing.T) {
	srv := (&giteatest.FakeAPI{
		Owner:   "owner",
		Project: "project",
		Issues:  []*gitea.Issue{testIssue()},
	}).NewServer(t)
	gi, backend := setupImporter(t, srv.URL)

	results := runImport(t, gi, backend)
	require.Empty(t, collectErrors(results))

	snap := onlyBug(t, backend).Snapshot()
	assertCreateMetadata := func(key, expected string) {
		t.Helper()
		got, ok := snap.GetCreateMetadata(key)
		require.True(t, ok, "missing create metadata key %s", key)
		assert.Equal(t, expected, got, "wrong value for create metadata key %s", key)
	}
	assertCreateMetadata(core.MetaKeyOrigin, target)
	assert.Equal(t, "gitea", target,
		"changing the Gitea target literal orphans existing bugs from the bridge")
	assertCreateMetadata(metaKeyGiteaID, "1")
	assertCreateMetadata(metaKeyGiteaOwner, "owner")
	assertCreateMetadata(metaKeyGiteaProject, "project")
	assertCreateMetadata(metaKeyGiteaBaseURL, srv.URL)
}

func TestImportThenExportThenImportNoDup(t *testing.T) {
	exporter := (&Gitea{}).NewExporter()
	if exporter == nil {
		t.Skip("Gitea exporter is not wired yet; enable this round-trip pin when NewExporter returns a real exporter")
	}

	t.Fatal("TODO: create a local bug, export it to Gitea, then import and assert len(AllIds()) == 1")
}

func TestImportMatchesByAllFiveMetadataKeys(t *testing.T) {
	testCases := []struct {
		name     string
		metadata map[string]string
		wantBugs int
	}{
		{
			name: "all keys match",
			metadata: map[string]string{
				core.MetaKeyOrigin:  target,
				metaKeyGiteaID:      "1",
				metaKeyGiteaOwner:   "owner",
				metaKeyGiteaProject: "project",
			},
			wantBugs: 1,
		},
		{
			name: "different origin",
			metadata: map[string]string{
				core.MetaKeyOrigin:  "github",
				metaKeyGiteaID:      "1",
				metaKeyGiteaOwner:   "owner",
				metaKeyGiteaProject: "project",
			},
			wantBugs: 2,
		},
		{
			name: "different gitea id",
			metadata: map[string]string{
				core.MetaKeyOrigin:  target,
				metaKeyGiteaID:      "2",
				metaKeyGiteaOwner:   "owner",
				metaKeyGiteaProject: "project",
			},
			wantBugs: 2,
		},
		{
			name: "different base url",
			metadata: map[string]string{
				core.MetaKeyOrigin:  target,
				metaKeyGiteaID:      "1",
				metaKeyGiteaOwner:   "owner",
				metaKeyGiteaProject: "project",
				metaKeyGiteaBaseURL: "https://elsewhere.invalid",
			},
			wantBugs: 2,
		},
		{
			name: "different owner",
			metadata: map[string]string{
				core.MetaKeyOrigin:  target,
				metaKeyGiteaID:      "1",
				metaKeyGiteaOwner:   "other-owner",
				metaKeyGiteaProject: "project",
			},
			wantBugs: 2,
		},
		{
			name: "different project",
			metadata: map[string]string{
				core.MetaKeyOrigin:  target,
				metaKeyGiteaID:      "1",
				metaKeyGiteaOwner:   "owner",
				metaKeyGiteaProject: "other-project",
			},
			wantBugs: 2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			srv := (&giteatest.FakeAPI{
				Owner:   "owner",
				Project: "project",
				Issues:  []*gitea.Issue{testIssue()},
			}).NewServer(t)
			gi, backend := setupImporter(t, srv.URL)

			metadata := map[string]string{
				core.MetaKeyOrigin:  tc.metadata[core.MetaKeyOrigin],
				metaKeyGiteaID:      tc.metadata[metaKeyGiteaID],
				metaKeyGiteaOwner:   tc.metadata[metaKeyGiteaOwner],
				metaKeyGiteaProject: tc.metadata[metaKeyGiteaProject],
				metaKeyGiteaBaseURL: srv.URL,
			}
			if baseURL, ok := tc.metadata[metaKeyGiteaBaseURL]; ok {
				metadata[metaKeyGiteaBaseURL] = baseURL
			}

			author, err := backend.Identities().NewRaw(
				"Existing User",
				"existing@example.com",
				"existing-user",
				"",
				nil,
				map[string]string{metaKeyGiteaLogin: "existing-user"},
			)
			require.NoError(t, err)
			_, _, err = backend.Bugs().NewRaw(
				author,
				time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC).Unix(),
				"existing issue",
				"existing body",
				nil,
				metadata,
			)
			require.NoError(t, err)

			results := runImport(t, gi, backend)
			require.Empty(t, collectErrors(results))
			assert.Len(t, backend.Bugs().AllIds(), tc.wantBugs,
				"matcher should only reuse the existing bug when all five metadata keys match")
		})
	}
}

func TestImportLabelCaseInsensitive(t *testing.T) {
	fa := &giteatest.FakeAPI{
		Owner:   "owner",
		Project: "project",
		Issues:  []*gitea.Issue{testIssue()},
	}
	srv := fa.NewServer(t)
	gi, backend := setupImporter(t, srv.URL)
	_ = runImport(t, gi, backend)

	b := onlyBug(t, backend)
	author := b.Snapshot().Author
	_, _, err := b.ChangeLabelsRaw(author, time.Date(2023, 1, 3, 0, 0, 0, 0, time.UTC).Unix(), []string{"bug"}, nil, nil)
	require.NoError(t, err)

	fa.Labels = []*gitea.Label{{ID: 1, Name: "Bug"}}
	_ = runImport(t, setupImporterOnExistingBackend(t, srv.URL, backend), backend)

	assert.Equal(t, []common.Label{common.Label("bug")}, onlyBug(t, backend).Snapshot().Labels,
		"label import should case-fold against existing local labels instead of adding Bug beside bug")
}

// TestImportLabelRenameByNameReconcilesImportedBug pins git-bug's name-based
// label model: when the remote issue's current label names change, the
// imported bug should match those names without trying to preserve a remote
// label identity that git-bug cannot represent globally.
func TestImportLabelRenameByNameReconcilesImportedBug(t *testing.T) {
	fa := &giteatest.FakeAPI{
		Owner:   "owner",
		Project: "project",
		Issues:  []*gitea.Issue{testIssue()},
		Labels:  []*gitea.Label{{ID: 1, Name: "bug"}},
	}
	srv := fa.NewServer(t)
	gi, backend := setupImporter(t, srv.URL)

	_ = runImport(t, gi, backend)

	fa.Labels = []*gitea.Label{{ID: 1, Name: "kind/bug"}}
	_ = runImport(t, setupImporterOnExistingBackend(t, srv.URL, backend), backend)

	assert.Equal(t, []common.Label{common.Label("kind/bug")}, onlyBug(t, backend).Snapshot().Labels,
		"remote label rename should reconcile the imported bug to the current upstream label names")
}

// TestImportLabelReconcileScopedToImportedBug verifies label reconciliation is
// scoped to the Gitea-imported bug. Matching label names on manual bugs or
// other forges are not global labels and must not be rewritten.
func TestImportLabelReconcileScopedToImportedBug(t *testing.T) {
	fa := &giteatest.FakeAPI{
		Owner:   "owner",
		Project: "project",
		Issues:  []*gitea.Issue{testIssue()},
		Labels:  []*gitea.Label{{ID: 1, Name: "bug"}},
	}
	srv := fa.NewServer(t)
	gi, backend := setupImporter(t, srv.URL)

	_ = runImport(t, gi, backend)

	manualAuthor, err := backend.Identities().NewRaw(
		"Manual User",
		"manual@example.com",
		"manual-user",
		"",
		nil,
		nil,
	)
	require.NoError(t, err)
	manualBug, _, err := backend.Bugs().NewRaw(
		manualAuthor,
		time.Date(2023, 1, 3, 0, 0, 0, 0, time.UTC).Unix(),
		"manual bug",
		"manual body",
		nil,
		nil,
	)
	require.NoError(t, err)
	_, _, err = manualBug.ChangeLabelsRaw(manualAuthor, time.Date(2023, 1, 3, 1, 0, 0, 0, time.UTC).Unix(), []string{"bug"}, nil, nil)
	require.NoError(t, err)

	otherForgeBug, _, err := backend.Bugs().NewRaw(
		manualAuthor,
		time.Date(2023, 1, 4, 0, 0, 0, 0, time.UTC).Unix(),
		"other forge bug",
		"other body",
		nil,
		map[string]string{core.MetaKeyOrigin: "github"},
	)
	require.NoError(t, err)
	_, _, err = otherForgeBug.ChangeLabelsRaw(manualAuthor, time.Date(2023, 1, 4, 1, 0, 0, 0, time.UTC).Unix(), []string{"bug"}, nil, nil)
	require.NoError(t, err)

	fa.Labels = []*gitea.Label{{ID: 1, Name: "kind/bug"}}
	_ = runImport(t, setupImporterOnExistingBackend(t, srv.URL, backend), backend)

	importedBug := onlyImportedGiteaBug(t, backend)
	assert.Equal(t, []common.Label{common.Label("kind/bug")}, importedBug.Snapshot().Labels,
		"imported Gitea bug should reconcile to current upstream label names")
	assert.Equal(t, []common.Label{common.Label("bug")}, manualBug.Snapshot().Labels,
		"manual bug labels should not be rewritten by Gitea label reconciliation")
	assert.Equal(t, []common.Label{common.Label("bug")}, otherForgeBug.Snapshot().Labels,
		"bugs imported from other forges should not be rewritten by Gitea label reconciliation")
}

func TestImportTitleChangeViaTypedComment(t *testing.T) {
	t.Skip("gitea.dev/sdk@v1.1.0 Comment lacks Type/OldTitle/NewTitle fields; add this pin when the SDK is upgraded")
}

func TestImportStateChangeViaTypedComment(t *testing.T) {
	t.Skip("gitea.dev/sdk@v1.1.0 Comment lacks Type fields for close/reopen system comments; add this pin when the SDK is upgraded")
}

func TestImportTitleUnsafeAfterCleanup(t *testing.T) {
	issue := testIssue()
	issue.Title = "\x1b"
	srv := (&giteatest.FakeAPI{
		Owner: "owner", Project: "project",
		Issues: []*gitea.Issue{issue},
	}).NewServer(t)
	gi, backend := setupImporter(t, srv.URL)

	var results []core.ImportResult
	require.NotPanics(t, func() {
		results = runImport(t, gi, backend)
	})
	errs := collectErrors(results)
	require.NotEmpty(t, errs)
	assert.Contains(t, errs[0].Error(), "title")
	assert.Empty(t, backend.Bugs().AllIds(),
		"invalid title import must not leave a half-created bug")
}

func TestImportBodyControlCharacters(t *testing.T) {
	issue := testIssue()
	issue.Body = "a\x00b\x1fc\n\tok"
	srv := (&giteatest.FakeAPI{
		Owner: "owner", Project: "project",
		Issues: []*gitea.Issue{issue},
	}).NewServer(t)
	gi, backend := setupImporter(t, srv.URL)

	results := runImport(t, gi, backend)
	require.Empty(t, collectErrors(results))

	snap := onlyBug(t, backend).Snapshot()
	require.NotEmpty(t, snap.Comments)
	assert.Equal(t, "abc\n\tok", snap.Comments[0].Message)
}

func TestImportVeryLongTitle(t *testing.T) {
	issue := testIssue()
	issue.Title = strings.Repeat("t", 10*1024)
	srv := (&giteatest.FakeAPI{
		Owner: "owner", Project: "project",
		Issues: []*gitea.Issue{issue},
	}).NewServer(t)
	gi, backend := setupImporter(t, srv.URL)

	var results []core.ImportResult
	require.NotPanics(t, func() {
		results = runImport(t, gi, backend)
	})

	errs := collectErrors(results)
	if len(errs) > 0 {
		assert.Empty(t, backend.Bugs().AllIds(),
			"failed long-title import should not leave a half-created bug")
		return
	}
	assert.Len(t, backend.Bugs().AllIds(), 1,
		"successful long-title import should commit exactly one bug")
}

func TestImportSinceSentOnIssuesRequest(t *testing.T) {
	fa := &giteatest.FakeAPI{
		Owner:   "owner",
		Project: "project",
		Issues:  []*gitea.Issue{testIssue()},
	}
	srv := fa.NewServer(t)
	gi, backend := setupImporter(t, srv.URL)

	since := time.Date(2023, 6, 1, 12, 0, 0, 0, time.UTC)
	_ = runImportSince(t, gi, backend, since)

	require.NotEmpty(t, fa.IssueRequests)
	got := fa.IssueRequests[0].URL.Query().Get("since")
	parsed, err := time.Parse(time.RFC3339, got)
	require.NoError(t, err)
	assert.Equal(t, since, parsed.UTC())
}

func TestImportSinceSentOnCommentsRequest(t *testing.T) {
	fa := &giteatest.FakeAPI{
		Owner:    "owner",
		Project:  "project",
		Issues:   []*gitea.Issue{testIssue()},
		Comments: []*gitea.Comment{testComment(1, "comment")},
	}
	srv := fa.NewServer(t)
	gi, backend := setupImporter(t, srv.URL)

	since := time.Date(2023, 6, 1, 12, 0, 0, 0, time.UTC)
	_ = runImportSince(t, gi, backend, since)

	require.NotEmpty(t, fa.CommentRequests)
	got := fa.CommentRequests[0].URL.Query().Get("since")
	parsed, err := time.Parse(time.RFC3339, got)
	require.NoError(t, err)
	assert.Equal(t, since, parsed.UTC())
}

func TestImportSinceSecondRunMovesForward(t *testing.T) {
	fa := &giteatest.FakeAPI{
		Owner:   "owner",
		Project: "project",
		Issues:  []*gitea.Issue{testIssue()},
	}
	srv := fa.NewServer(t)
	gi, backend := setupImporter(t, srv.URL)

	_ = runImport(t, gi, backend)
	b := onlyBug(t, backend)
	opsBefore := len(b.Snapshot().Operations)

	since := time.Date(2023, 6, 1, 12, 0, 0, 0, time.UTC)
	_ = runImportSince(t, setupImporterOnExistingBackend(t, srv.URL, backend), backend, since)

	got := fa.IssueRequests[len(fa.IssueRequests)-1].URL.Query().Get("since")
	parsed, err := time.Parse(time.RFC3339, got)
	require.NoError(t, err)
	assert.Equal(t, since, parsed.UTC())
	assert.Equal(t, opsBefore, len(onlyBug(t, backend).Snapshot().Operations),
		"incremental re-import of unchanged items should not add operations")
}

func TestImportErrorCarriesBugID(t *testing.T) {
	comment := testComment(1, "bad author")
	comment.Poster = &gitea.User{UserName: "flaky-commenter"}
	srv := (&giteatest.FakeAPI{
		Owner:        "owner",
		Project:      "project",
		Issues:       []*gitea.Issue{testIssue()},
		Comments:     []*gitea.Comment{comment},
		UserErrLogin: "flaky-commenter",
	}).NewServer(t)

	gi, backend := setupImporter(t, srv.URL)
	results := runImport(t, gi, backend)

	b := onlyBug(t, backend)
	var errorResult *core.ImportResult
	for i := range results {
		if results[i].Err != nil {
			errorResult = &results[i]
			break
		}
	}
	require.NotNil(t, errorResult, "comment author lookup failure should emit an ImportError")
	assert.Equal(t, b.Id(), errorResult.EntityId,
		"ImportError after bug creation should carry the affected bug id")
}

func TestImportSkipsUserAPIWhenCached(t *testing.T) {
	user := &gitea.User{UserName: "alice", FullName: "Alice", Email: "alice@example.com"}
	fa := &giteatest.FakeAPI{
		Owner:   "owner",
		Project: "project",
		Issues:  []*gitea.Issue{{ID: 1, Index: 1, Title: "first", Body: "b", Poster: user, Created: time.Now()}},
	}
	srv := fa.NewServer(t)
	gi, backend := setupImporter(t, srv.URL)
	_ = runImport(t, gi, backend)

	firstCount := len(fa.UserRequests)
	require.Equal(t, 1, firstCount)

	fa.Issues = []*gitea.Issue{{ID: 2, Index: 2, Title: "second", Body: "b", Poster: user, Created: time.Now()}}
	_ = runImport(t, setupImporterOnExistingBackend(t, srv.URL, backend), backend)

	assert.Equal(t, firstCount, len(fa.UserRequests),
		"cached identity for alice should avoid another /users/alice request")
}

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

	// A comment-listing failure must not be hidden behind a successful partial
	// issue import.
	assert.NotEmpty(t, collectErrors(results), "expected ImportError when comment fetch fails")
}

func TestImportStopsOnCommentError(t *testing.T) {
	ts := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	user := &gitea.User{UserName: "testuser", FullName: "Test User", Email: "u@example.com"}
	srv := (&giteatest.FakeAPI{
		Owner:          "owner",
		Project:        "project",
		CommentErrPage: 1,
		Issues: []*gitea.Issue{
			{ID: 1, Index: 1, Title: "first", Body: "b1", Poster: user, Created: ts},
			{ID: 2, Index: 2, Title: "second should not import", Body: "b2", Poster: user, Created: ts},
		},
		CommentsByIssue: map[int64][]*gitea.Comment{
			1: {{ID: 11, Body: "comment fetch fails before this matters", Poster: user, Created: ts, Updated: ts}},
			2: {{ID: 21, Body: "should not be fetched", Poster: user, Created: ts, Updated: ts}},
		},
	}).NewServer(t)
	gi, backend := setupImporter(t, srv.URL)

	results := runImport(t, gi, backend)

	// This pins the current fail-fast behavior for comment iterator errors.
	// If Gitea import switches to continue-on-error, update this expectation.
	require.NotEmpty(t, collectErrors(results),
		"comment fetch failure on issue #1 should surface as an ImportError")
	assert.Len(t, backend.Bugs().AllIds(), 1,
		"current behavior: import stops after a comment iterator error; issue #2 is never imported")
}

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

	// Some SDK versions return a nil *User with a 404 response; the importer
	// should still create a placeholder identity for the requested login.
	results := runImport(t, gi, backend)
	assert.Empty(t, collectErrors(results),
		"a 404 on user lookup should fall back to a placeholder identity, not panic")

	_, err := backend.Identities().ResolveIdentityImmutableMetadata(metaKeyGiteaLogin, "deleted-user")
	assert.NoError(t, err,
		"a placeholder identity tagged with the looked-up login should exist")
}

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
			Poster:  &gitea.User{UserName: "testuser", FullName: "Test User", Email: "u@example.com"},
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

func TestImportRepoNotFound(t *testing.T) {
	srv := (&giteatest.FakeAPI{
		Owner:        "wrong-owner",
		Project:      "wrong-project",
		RepoNotFound: true,
	}).NewServer(t)

	repo := repository.CreateGoGitTestRepo(t, false)
	token := auth.NewToken(target, "test-token")
	token.SetMetadata(auth.MetaKeyLogin, "testuser")
	token.SetMetadata(auth.MetaKeyBaseURL, srv.URL)
	require.NoError(t, auth.Store(repo, token))

	backend, err := cache.NewRepoCacheNoEvents(repo)
	require.NoError(t, err)
	t.Cleanup(func() { _ = backend.Close() })

	gi := &giteaImporter{}
	// Ask for a repo the fake server intentionally does not register.
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
	// and reached a send. Anything emitted by the import is fine here; for the
	// first issue, ImportEventIdentity comes before ImportEventBug.
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

func TestImportClosesOutputChannel(t *testing.T) {
	srv := (&giteatest.FakeAPI{
		Owner:   "owner",
		Project: "project",
		Issues:  []*gitea.Issue{testIssue()},
	}).NewServer(t)
	gi, backend := setupImporter(t, srv.URL)

	ch, err := gi.ImportAll(context.Background(), backend, time.Time{})
	require.NoError(t, err)

	// Most importer tests range over this channel; make the close contract
	// explicit so a refactor fails here instead of deadlocking the suite.
	for range ch {
	}

	select {
	case _, ok := <-ch:
		assert.False(t, ok, "channel should be closed after ImportAll completes")
	case <-time.After(time.Second):
		t.Fatal("channel was not closed within 1s of draining — ImportAll likely missing `defer close`")
	}
}
