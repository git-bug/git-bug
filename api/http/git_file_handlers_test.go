package http

import (
	"bytes"
	"context"
	"image"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/api/auth"
	"github.com/git-bug/git-bug/cache"
	"github.com/git-bug/git-bug/repository"
)

func TestGitFileHandlers(t *testing.T) {
	repo := repository.CreateGoGitTestRepo(t, false)

	mrc := cache.NewMultiRepoCache()
	repoCache, events := mrc.RegisterDefaultRepository(repo)
	for event := range events {
		require.NoError(t, event.Err)
	}

	author, err := repoCache.Identities().New("test identity", "test@test.org")
	require.NoError(t, err)
	err = repoCache.SetUserIdentity(author)
	require.NoError(t, err)

	// Build a PNG image to use as test content.
	img := image.NewNRGBA(image.Rect(0, 0, 50, 50))
	data := &bytes.Buffer{}
	err = png.Encode(data, img)
	require.NoError(t, err)
	imgBytes := data.Bytes()

	// ── Upload ────────────────────────────────────────────────────────────────

	t.Run("Upload", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, err := writer.CreateFormFile("uploadfile", "noname")
		require.NoError(t, err)
		_, err = part.Write(imgBytes)
		require.NoError(t, err)
		require.NoError(t, writer.Close())

		w := httptest.NewRecorder()
		r, _ := http.NewRequest("POST", "/", body)
		r.Header.Add("Content-Type", writer.FormDataContentType())
		r = r.WithContext(auth.CtxWithUser(r.Context(), author.Id()))
		r = mux.SetURLVars(r, map[string]string{"repo": ""})

		NewGitUploadFileHandler(mrc).ServeHTTP(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, `{"hash":"3426a1488292d8f3f3c59ca679681336542b986f"}`, w.Body.String())
	})

	// ── Download by hash ──────────────────────────────────────────────────────

	t.Run("DownloadByHash", func(t *testing.T) {
		w := httptest.NewRecorder()
		r, _ := http.NewRequest("GET", "/", nil)
		r = r.WithContext(auth.CtxWithUser(r.Context(), author.Id()))
		r = mux.SetURLVars(r, map[string]string{
			"repo": "",
			"rest": "3426a1488292d8f3f3c59ca679681336542b986f",
		})

		NewGitFileHandler(mrc).ServeHTTP(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "image/png", w.Header().Get("Content-Type"))
		assert.Equal(t, imgBytes, w.Body.Bytes())
	})

	// Set up commits to test ref+path resolution.
	//
	// Ambiguity test: git's filesystem prevents refs/heads/feature and
	// refs/heads/feature/foo from coexisting (file vs directory). Instead, use
	// a branch "feature" and a tag "feature/foo" — different namespaces, no
	// conflict. resolveRefToHash checks both heads and tags, so the tag is a
	// valid ref. With longest-ref-first resolution, "feature/foo/image.png"
	// must resolve via the tag (imgBytes), not via the branch (which has no
	// foo/image.png and would 404).
	imgHash, err := repo.StoreData(imgBytes)
	require.NoError(t, err)

	otherBytes := []byte("other content")
	otherHash, err := repo.StoreData(otherBytes)
	require.NoError(t, err)

	imgTreeHash, err := repo.StoreTree([]repository.TreeEntry{
		{ObjectType: repository.Blob, Hash: imgHash, Name: "image.png"},
	})
	require.NoError(t, err)

	otherTreeHash, err := repo.StoreTree([]repository.TreeEntry{
		{ObjectType: repository.Blob, Hash: otherHash, Name: "image.png"},
	})
	require.NoError(t, err)

	mainCommit, err := repo.StoreCommit(imgTreeHash)
	require.NoError(t, err)
	featureCommit, err := repo.StoreCommit(otherTreeHash)
	require.NoError(t, err)

	require.NoError(t, repo.UpdateRef("refs/heads/main", mainCommit))
	// "feature" branch has otherBytes; "feature/foo" tag has imgBytes.
	require.NoError(t, repo.UpdateRef("refs/heads/feature", featureCommit))
	require.NoError(t, repo.UpdateRef("refs/tags/feature/foo", mainCommit))

	handler := NewGitFileHandler(mrc)
	authCtx := auth.CtxWithUser(context.Background(), author.Id())

	serve := func(rest string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		r, _ := http.NewRequest("GET", "/", nil)
		r = r.WithContext(authCtx)
		r = mux.SetURLVars(r, map[string]string{"repo": "", "rest": rest})
		handler.ServeHTTP(w, r)
		return w
	}

	serveWithHeader := func(rest, headerName, headerVal string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		r, _ := http.NewRequest("GET", "/", nil)
		r = r.WithContext(authCtx)
		r.Header.Set(headerName, headerVal)
		r = mux.SetURLVars(r, map[string]string{"repo": "", "rest": rest})
		handler.ServeHTTP(w, r)
		return w
	}

	// ── Download by ref+path (simple ref) ─────────────────────────────────────

	t.Run("DownloadByRefPath", func(t *testing.T) {
		w := serve("main/image.png")
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "image/png", w.Header().Get("Content-Type"))
		assert.Equal(t, imgBytes, w.Body.Bytes())
	})

	// ── Download by ref+path (ref with slash) ─────────────────────────────────

	t.Run("DownloadByRefWithSlash", func(t *testing.T) {
		// "feature/foo" is a tag; verify multi-segment refs resolve correctly.
		w := serve("feature/foo/image.png")
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "image/png", w.Header().Get("Content-Type"))
		assert.Equal(t, imgBytes, w.Body.Bytes())
	})

	// ── Ambiguous ref: longest ref wins ───────────────────────────────────────
	// Both "feature" (branch, otherBytes) and "feature/foo" (tag, imgBytes)
	// exist. "feature/foo/image.png" must resolve via the longer ref
	// "feature/foo" → imgBytes, not via "feature" → foo/image.png (404).

	t.Run("AmbiguousRefLongestWins", func(t *testing.T) {
		w := serve("feature/foo/image.png")
		assert.Equal(t, http.StatusOK, w.Code)
		// Must be imgBytes (from tag feature/foo), not otherBytes (from branch feature).
		assert.Equal(t, imgBytes, w.Body.Bytes())
	})

	// ── Conditional GET: 304 with matching ETag ───────────────────────────────

	t.Run("ConditionalGet304", func(t *testing.T) {
		// First request to get the ETag.
		w := serve("main/image.png")
		require.Equal(t, http.StatusOK, w.Code)
		etag := w.Header().Get("ETag")
		require.NotEmpty(t, etag)

		// Second request with If-None-Match should get 304.
		w = serveWithHeader("main/image.png", "If-None-Match", etag)
		assert.Equal(t, http.StatusNotModified, w.Code)
		assert.Equal(t, etag, w.Header().Get("ETag"))
		assert.Empty(t, w.Body.Bytes())
	})

	t.Run("ConditionalGetWeakETag", func(t *testing.T) {
		w := serve("main/image.png")
		require.Equal(t, http.StatusOK, w.Code)
		etag := w.Header().Get("ETag")

		// Weak form of the same ETag should also match.
		w = serveWithHeader("main/image.png", "If-None-Match", "W/"+etag)
		assert.Equal(t, http.StatusNotModified, w.Code)
	})

	t.Run("ConditionalGetWildcard", func(t *testing.T) {
		w := serveWithHeader("main/image.png", "If-None-Match", "*")
		assert.Equal(t, http.StatusNotModified, w.Code)
	})

	// ── Not found ─────────────────────────────────────────────────────────────

	t.Run("NotFound", func(t *testing.T) {
		w := serve("main/nonexistent.png")
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	// ── Malformed: single segment that is not a hash ───────────────────────────

	t.Run("MalformedSingleSegment", func(t *testing.T) {
		w := serve("main")
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}
