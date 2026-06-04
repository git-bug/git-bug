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

	assert.NotEmpty(t, collectErrors(results), "expected ImportError for null Poster comment, not panic")
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
