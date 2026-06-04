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

	// CommentsByIssue/LabelsByIssue, if set, override the singular Comments/Labels
	// fields and let tests configure per-issue responses (keyed by Issue.Index).
	CommentsByIssue map[int64][]*gitea.Comment
	LabelsByIssue   map[int64][]*gitea.Label

	// NotFoundUsers is a list of usernames that return 404 from userGet.
	NotFoundUsers []string

	// UserErrLogin, if set, makes the user endpoint return 500 for that login.
	UserErrLogin string

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

func parsePagination(r *http.Request) (page, limit int) {
	page, limit = 1, 10
	if p := r.URL.Query().Get("page"); p != "" {
		if n, err := strconv.Atoi(p); err == nil && n > 0 {
			page = n
		}
	}
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}
	return
}

func pageSlice(page, limit, total int) (start, end int) {
	start = (page - 1) * limit
	end = start + limit
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}
	return
}

func (fa *FakeAPI) commentsFor(idx int64) []*gitea.Comment {
	if c, ok := fa.CommentsByIssue[idx]; ok {
		return c
	}
	if idx == 1 {
		return fa.Comments
	}
	return nil
}

func (fa *FakeAPI) labelsFor(idx int64) []*gitea.Label {
	if l, ok := fa.LabelsByIssue[idx]; ok {
		return l
	}
	if idx == 1 {
		return fa.Labels
	}
	return nil
}

// NewServer starts an httptest.Server backed by this FakeAPI and registers
// a cleanup to close it when t finishes.
func (fa *FakeAPI) NewServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	issuesPath := fmt.Sprintf("/api/v1/repos/%s/%s/issues", fa.Owner, fa.Project)
	issuesPrefix := issuesPath + "/"

	// https://codeberg.org/api/swagger#/issue/issueListIssues
	mux.HandleFunc(issuesPath, func(w http.ResponseWriter, r *http.Request) {
		fa.IssueRequests = append(fa.IssueRequests, r)
		page, limit := parsePagination(r)

		if fa.IssueErrPage > 0 && page == fa.IssueErrPage {
			http.Error(w, "simulated server error", http.StatusInternalServerError)
			return
		}

		// X-Total-Count is stable but undocumented;
		// see https://codeberg.org/forgejo/forgejo/issues/12931
		w.Header().Set("X-Total-Count", strconv.Itoa(len(fa.Issues)))

		start, end := pageSlice(page, limit, len(fa.Issues))
		writeJSON(w, fa.Issues[start:end])
	})

	// Sub-paths under /issues/{index}/ — comments and labels are dispatched
	// here so tests can configure responses per-issue.
	mux.HandleFunc(issuesPrefix, func(w http.ResponseWriter, r *http.Request) {
		rest := strings.TrimPrefix(r.URL.Path, issuesPrefix)
		parts := strings.SplitN(rest, "/", 2)
		if len(parts) < 2 {
			http.NotFound(w, r)
			return
		}
		idx, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			http.NotFound(w, r)
			return
		}

		page, limit := parsePagination(r)

		switch parts[1] {
		case "comments":
			fa.CommentRequests = append(fa.CommentRequests, r)
			if fa.CommentErrPage > 0 && page == fa.CommentErrPage {
				http.Error(w, "simulated server error", http.StatusInternalServerError)
				return
			}
			comments := fa.commentsFor(idx)
			w.Header().Set("X-Total-Count", strconv.Itoa(len(comments)))
			start, end := pageSlice(page, limit, len(comments))
			writeJSON(w, comments[start:end])
		case "labels":
			if fa.LabelErrPage > 0 && page == fa.LabelErrPage {
				http.Error(w, "simulated server error", http.StatusInternalServerError)
				return
			}
			labels := fa.labelsFor(idx)
			start, end := pageSlice(page, limit, len(labels))
			writeJSON(w, labels[start:end])
		default:
			http.NotFound(w, r)
		}
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
		if fa.UserErrLogin != "" && login == fa.UserErrLogin {
			http.Error(w, "simulated server error", http.StatusInternalServerError)
			return
		}
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
