package http

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/MichaelMure/git-bug/api/auth"
	"github.com/MichaelMure/git-bug/cache"
)

// implement a http.Handler that will accept and store content into git blob.
//
// Expected gorilla/mux parameters:
//   - "repo" : the ref of the repo or "" for the default one
type gitUploadFileHandler struct {
	mrc *cache.MultiRepoCache
}

func NewGitUploadFileHandler(mrc *cache.MultiRepoCache) http.Handler {
	return &gitUploadFileHandler{mrc: mrc}
}

func (gufh *gitUploadFileHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	var repo *cache.RepoCache
	var err error

	repoVar := mux.Vars(r)["repo"]
	switch repoVar {
	case "":
		repo, err = gufh.mrc.DefaultRepo()
	default:
		repo, err = gufh.mrc.ResolveRepo(repoVar)
	}

	if err != nil {
		http.Error(rw, "invalid repo reference", http.StatusBadRequest)
		return
	}

	_, err = auth.UserFromCtx(r.Context(), repo)
	if err == auth.ErrNotAuthenticated {
		http.Error(rw, "read-only mode or not logged in", http.StatusForbidden)
		return
	} else if err != nil {
		http.Error(rw, fmt.Sprintf("loading identity: %v", err), http.StatusInternalServerError)
		return
	}

	// 100MB (github limit)
	var maxUploadSize int64 = 100 * 1000 * 1000
	r.Body = http.MaxBytesReader(rw, r.Body, maxUploadSize)
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		http.Error(rw, "file too big (100MB max)", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("uploadfile")
	if err != nil {
		http.Error(rw, "invalid file", http.StatusBadRequest)
		return
	}
	defer file.Close()
	fileBytes, err := io.ReadAll(file)
	if err != nil {
		http.Error(rw, "invalid file", http.StatusBadRequest)
		return
	}

	filetype := http.DetectContentType(fileBytes)
	if filetype != "image/jpeg" && filetype != "image/jpg" &&
		filetype != "image/gif" && filetype != "image/png" {
		http.Error(rw, "invalid file type", http.StatusBadRequest)
		return
	}

	hash, err := repo.StoreData(fileBytes)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	type response struct {
		Hash string `json:"hash"`
	}

	resp := response{Hash: string(hash)}

	js, err := json.Marshal(resp)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	rw.Header().Set("Content-Type", "application/json")
	_, err = rw.Write(js)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
}
