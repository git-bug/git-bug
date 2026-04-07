package http

import (
	"bytes"
	"errors"
	"io"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/git-bug/git-bug/cache"
	"github.com/git-bug/git-bug/repository"
)

// implement a http.Handler that will read and server git blob.
//
// Expected gorilla/mux parameters:
//   - "repo" : the ref of the repo or "" for the default one
//   - "hash" : the git hash of the file to retrieve
type gitFileHandler struct {
	mrc *cache.MultiRepoCache
}

func NewGitFileHandler(mrc *cache.MultiRepoCache) http.Handler {
	return &gitFileHandler{mrc: mrc}
}

func (gfh *gitFileHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	var repo *cache.RepoCache
	var err error

	repoVar := mux.Vars(r)["repo"]
	switch repoVar {
	case "":
		repo, err = gfh.mrc.DefaultRepo()
	default:
		repo, err = gfh.mrc.ResolveRepo(repoVar)
	}

	if err != nil {
		http.Error(rw, "invalid repo reference", http.StatusBadRequest)
		return
	}

	hash := repository.Hash(mux.Vars(r)["hash"])
	if !hash.IsValid() {
		http.Error(rw, "invalid git hash", http.StatusBadRequest)
		return
	}

	reader, err := repo.ReadData(hash)
	if errors.Is(err, repository.ErrNotFound) {
		http.Error(rw, "blob not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	defer func() {
		_ = reader.Close()
	}()

	serveContent(rw, r, reader)
}

// serveContent is a somewhat equivalent of http.serveContent, without support for range request.
// This is necessary as the repo (and go-git)'s data reader doesn't support Seek().
func serveContent(w http.ResponseWriter, r *http.Request, content io.Reader) {
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

	w.WriteHeader(http.StatusOK)
	if r.Method == http.MethodHead {
		return
	}

	_, _ = io.Copy(w, content)
}
