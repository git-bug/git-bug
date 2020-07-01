package http

import (
	"bytes"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/repository"
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

	// TODO: this mean that the whole file will he buffered in memory
	// This can be a problem for big files. There might be a way around
	// that by implementing a io.ReadSeeker that would read and discard
	// data when a seek is called.
	data, err := repo.ReadData(hash)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	http.ServeContent(rw, r, "", time.Now(), bytes.NewReader(data))
}
