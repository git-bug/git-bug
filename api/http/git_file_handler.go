package http

import (
	"io"
	"net/http"
	"strings"

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
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	defer func() {
		_ = reader.Close()
	}()

	ServeContent(rw, r, reader)
}

// ServeContent is a somewhat equivalent of http.ServeContent, without support for range request.
// This is necessary as the repo (and go-git)'s data reader doesn't support Seek().
func ServeContent(w http.ResponseWriter, r *http.Request, content io.Reader) {
	if w.Header().Get("Content-Type") == "" {
		// Sniff the type from the first up to 512 bytes.
		var buf [512]byte
		n, err := io.ReadFull(content, buf[:])
		switch err {
		case nil:
			w.Header().Set("Content-Type", http.DetectContentType(buf[:n]))
			content = io.MultiReader(strings.NewReader(string(buf[:n])), content)
		case io.ErrUnexpectedEOF, io.EOF:
			w.Header().Set("Content-Type", http.DetectContentType(buf[:n]))
			content = strings.NewReader(string(buf[:n]))
		default:
			// If sniffing fails unexpectedly, fall back safely.
			w.Header().Set("Content-Type", "application/octet-stream")
		}
	}

	w.WriteHeader(http.StatusOK)
	if r.Method == http.MethodHead {
		return
	}

	_, _ = io.Copy(w, content)
}
