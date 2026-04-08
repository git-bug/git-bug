package http

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"

	"github.com/git-bug/git-bug/cache"
	"github.com/git-bug/git-bug/repository"
)

// gitFileHandler implements an http.Handler that reads and serves a git blob.
//
// The route is /gitfile/{repo}/{rest:.+} where rest is resolved as follows:
//   - If rest contains no slash and is a valid git hash: serve the blob by hash.
//   - Otherwise: try each slash as a split point between ref and path, longest
//     ref first (right to left). This handles refs with slashes (e.g.
//     "feature/foo") and matches the ref that is most specific.
//
// Expected gorilla/mux parameters:
//   - "repo": the repo ref or "" for the default one
//   - "rest": the hash, or the ref+path combined
type gitFileHandler struct {
	mrc *cache.MultiRepoCache
}

func NewGitFileHandler(mrc *cache.MultiRepoCache) http.Handler {
	return &gitFileHandler{mrc: mrc}
}

func (gfh *gitFileHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	var repo *cache.RepoCache
	var err error

	switch repoVar := vars["repo"]; repoVar {
	case "":
		repo, err = gfh.mrc.DefaultRepo()
	default:
		repo, err = gfh.mrc.ResolveRepo(repoVar)
	}
	if err != nil {
		http.Error(rw, "invalid repo reference", http.StatusBadRequest)
		return
	}

	rest := vars["rest"]
	if rest == "" {
		http.Error(rw, "missing path", http.StatusBadRequest)
		return
	}

	// If rest is a single segment that is a valid git hash, serve by hash.
	if !strings.Contains(rest, "/") {
		if hash := repository.Hash(rest); hash.IsValid() {
			reader, err := repo.ReadData(hash)
			if errors.Is(err, repository.ErrNotFound) {
				http.Error(rw, "not found", http.StatusNotFound)
				return
			}
			if err != nil {
				http.Error(rw, "internal server error", http.StatusInternalServerError)
				return
			}
			defer reader.Close()
			serveContent(rw, r, reader, -1, hash)
			return
		}
		// Single segment that is not a hash: malformed request.
		http.Error(rw, "expected <hash> or <ref>/<path>", http.StatusBadRequest)
		return
	}

	// Greedy ref+path resolution: try split points longest ref first (right to
	// left) so that refs with slashes (e.g. "feature/foo") take precedence over
	// shorter prefixes.
	segments := strings.Split(rest, "/")
	for i := len(segments) - 1; i >= 1; i-- {
		ref := strings.Join(segments[:i], "/")
		path := strings.Join(segments[i:], "/")
		rc, size, hash, err := repo.BrowseRepo().BlobAtPath(ref, path)
		if errors.Is(err, repository.ErrNotFound) {
			continue
		}
		if err != nil {
			http.Error(rw, "internal server error", http.StatusInternalServerError)
			return
		}
		defer rc.Close()
		serveContent(rw, r, rc, size, hash)
		return
	}

	http.Error(rw, "not found", http.StatusNotFound)
}

// ifNoneMatchContains reports whether the If-None-Match header value matches
// etag per RFC 9110 weak comparison: handles "*", comma-separated lists, and
// weak validators (W/"...").
func ifNoneMatchContains(header, etag string) bool {
	if header == "*" {
		return true
	}
	for _, token := range strings.Split(header, ",") {
		token = strings.TrimSpace(token)
		token = strings.TrimPrefix(token, "W/")
		if token == etag {
			return true
		}
	}
	return false
}

// serveContent is a somewhat equivalent of http.serveContent, without support for range request.
// This is necessary as the repo (and go-git)'s data reader doesn't support Seek().
func serveContent(w http.ResponseWriter, r *http.Request, content io.Reader, size int64, hash repository.Hash) {
	if hash.IsValid() {
		etag := `"` + string(hash) + `"`
		w.Header().Set("ETag", etag)
		if ifNoneMatchContains(r.Header.Get("If-None-Match"), etag) {
			w.WriteHeader(http.StatusNotModified)
			return
		}
	}

	if w.Header().Get("Content-Type") == "" {
		// Sniff the type from the first up to 512 bytes.
		var buf [512]byte
		n, err := io.ReadFull(content, buf[:])
		switch err {
		case nil:
			w.Header().Set("Content-Type", http.DetectContentType(buf[:n]))
			content = io.MultiReader(bytes.NewReader(buf[:n]), content)
		case io.ErrUnexpectedEOF, io.EOF:
			w.Header().Set("Content-Type", http.DetectContentType(buf[:n]))
			content = bytes.NewReader(buf[:n])
		default:
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
	}

	if size >= 0 {
		w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
	}

	w.WriteHeader(http.StatusOK)
	if r.Method == http.MethodHead {
		return
	}

	_, _ = io.Copy(w, content)
}
