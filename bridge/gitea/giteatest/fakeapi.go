// Package giteatest provides a fake Forgejo/Gitea API server for use in tests.
package giteatest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	gitea "gitea.dev/sdk"
)

// FakeAPI is a configurable httptest server mimicking the Forgejo/Gitea API.
// Set fields before calling NewServer; the handlers read them at request time.
type FakeAPI struct {
	Owner   string
	Project string

	Issues         []*gitea.Issue
	Comments       []*gitea.Comment
	CommentErrPage int // page number that returns 500; 0 = never
	IssueErrPage   int // page number that returns 500 on issues endpoint; 0 = never
	LabelErrPage   int // page number that returns 500 on labels endpoint; 0 = never

	Labels []*gitea.Label

	// NotFoundUsers is a list of usernames that return 404 from userGet.
	NotFoundUsers []string

	// RepoNotFound makes the repo endpoint return 404.
	RepoNotFound bool

	// IssueRequests accumulates every request made to the issues endpoint.
	// Inspect after running to verify query parameters such as `since`.
	IssueRequests []*http.Request

	// CommentRequests accumulates every request made to the comments endpoint.
	CommentRequests []*http.Request
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

// NewServer starts an httptest.Server backed by this FakeAPI and registers
// a cleanup to close it when t finishes.
func (fa *FakeAPI) NewServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	issuesPath := fmt.Sprintf("/api/v1/repos/%s/%s/issues", fa.Owner, fa.Project)
	commentsPath := fmt.Sprintf("/api/v1/repos/%s/%s/issues/1/comments", fa.Owner, fa.Project)
	labelsPath := fmt.Sprintf("/api/v1/repos/%s/%s/issues/1/labels", fa.Owner, fa.Project)

	// https://codeberg.org/api/swagger#/issue/issueListIssues
	mux.HandleFunc(issuesPath, func(w http.ResponseWriter, r *http.Request) {
		fa.IssueRequests = append(fa.IssueRequests, r)

		page := 1
		if p := r.URL.Query().Get("page"); p != "" {
			if n, err := strconv.Atoi(p); err == nil && n > 0 {
				page = n
			}
		}
		limit := 10
		if l := r.URL.Query().Get("limit"); l != "" {
			if n, err := strconv.Atoi(l); err == nil && n > 0 {
				limit = n
			}
		}

		if fa.IssueErrPage > 0 && page == fa.IssueErrPage {
			http.Error(w, "simulated server error", http.StatusInternalServerError)
			return
		}

		// X-Total-Count is stable but undocumented;
		// see https://codeberg.org/forgejo/forgejo/issues/12931
		w.Header().Set("X-Total-Count", strconv.Itoa(len(fa.Issues)))

		start := (page - 1) * limit
		end := start + limit
		if start > len(fa.Issues) {
			start = len(fa.Issues)
		}
		if end > len(fa.Issues) {
			end = len(fa.Issues)
		}
		writeJSON(w, fa.Issues[start:end])
	})

	// https://codeberg.org/api/swagger#/issue/issueGetComments
	mux.HandleFunc(commentsPath, func(w http.ResponseWriter, r *http.Request) {
		fa.CommentRequests = append(fa.CommentRequests, r)
		page := 1
		if p := r.URL.Query().Get("page"); p != "" {
			if n, err := strconv.Atoi(p); err == nil && n > 0 {
				page = n
			}
		}
		limit := 10
		if l := r.URL.Query().Get("limit"); l != "" {
			if n, err := strconv.Atoi(l); err == nil && n > 0 {
				limit = n
			}
		}

		if fa.CommentErrPage > 0 && page == fa.CommentErrPage {
			http.Error(w, "simulated server error", http.StatusInternalServerError)
			return
		}

		start := (page - 1) * limit
		end := start + limit
		if end > len(fa.Comments) {
			end = len(fa.Comments)
		}

		writeJSON(w, fa.Comments[start:end])
	})

	// https://codeberg.org/api/swagger#/issue/issueGetLabels
	mux.HandleFunc(labelsPath, func(w http.ResponseWriter, r *http.Request) {
		page := 1
		if p := r.URL.Query().Get("page"); p != "" {
			if n, err := strconv.Atoi(p); err == nil && n > 0 {
				page = n
			}
		}
		limit := 10
		if l := r.URL.Query().Get("limit"); l != "" {
			if n, err := strconv.Atoi(l); err == nil && n > 0 {
				limit = n
			}
		}

		if fa.LabelErrPage > 0 && page == fa.LabelErrPage {
			http.Error(w, "simulated server error", http.StatusInternalServerError)
			return
		}

		start := (page - 1) * limit
		end := start + limit
		if start > len(fa.Labels) {
			start = len(fa.Labels)
		}
		if end > len(fa.Labels) {
			end = len(fa.Labels)
		}
		writeJSON(w, fa.Labels[start:end])
	})

	// https://codeberg.org/api/swagger#/miscellaneous/getVersion
	mux.HandleFunc("/api/v1/version", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]string{"version": "1.24.0"})
	})

	// https://codeberg.org/api/swagger#/repository/repoGet
	repoPath := fmt.Sprintf("/api/v1/repos/%s/%s", fa.Owner, fa.Project)
	mux.HandleFunc(repoPath, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != repoPath {
			http.NotFound(w, r)
			return
		}
		if fa.RepoNotFound {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		writeJSON(w, &gitea.Repository{Name: fa.Project, Owner: &gitea.User{UserName: fa.Owner}})
	})

	// https://codeberg.org/api/swagger#/user/userGet
	mux.HandleFunc("/api/v1/users/", func(w http.ResponseWriter, r *http.Request) {
		login := strings.TrimPrefix(r.URL.Path, "/api/v1/users/")
		for _, notFound := range fa.NotFoundUsers {
			if notFound == login {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
		}
		writeJSON(w, &gitea.User{UserName: login, FullName: login, Email: login + "@example.com"})
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}
