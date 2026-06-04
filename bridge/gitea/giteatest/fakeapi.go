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
	"time"

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

	Labels     []*gitea.Label
	RepoLabels []*gitea.Label

	// CommentsByIssue/LabelsByIssue, if set, override the singular Comments/Labels
	// fields and let tests configure per-issue responses (keyed by Issue.Index).
	CommentsByIssue map[int64][]*gitea.Comment
	LabelsByIssue   map[int64][]*gitea.Label

	// NotFoundUsers is a list of usernames that return 404 from userGet.
	NotFoundUsers []string

	// UserErrLogin, if set, makes the user endpoint return 500 for that login.
	UserErrLogin string

	// UserNetworkErrLogin, if set, makes the user endpoint abort the
	// connection without writing a response when that login is requested.
	// Simulates a network-level failure where the client gets (nil, err).
	UserNetworkErrLogin string

	// RepoNotFound makes the repo endpoint return 404.
	RepoNotFound bool

	// RepoErrStatus, if non-zero, makes the repo endpoint return that HTTP
	// status (e.g. 500). Takes precedence over RepoNotFound.
	RepoErrStatus int

	// IssueRequests accumulates every request made to the issues endpoint.
	// Inspect after running to verify query parameters such as `since`.
	IssueRequests []*http.Request

	// CommentRequests accumulates every request made to the comments endpoint.
	CommentRequests []*http.Request

	// UserRequests accumulates every request made to the users endpoint.
	UserRequests []*http.Request

	nextIssueID    int64
	nextIssueIndex int64
	nextCommentID  int64
	nextLabelID    int64
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

func decodeJSON[T any](w http.ResponseWriter, r *http.Request) (T, bool) {
	var v T
	if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return v, false
	}
	return v, true
}

func (fa *FakeAPI) initSequences() {
	for _, issue := range fa.Issues {
		if issue.ID > fa.nextIssueID {
			fa.nextIssueID = issue.ID
		}
		if issue.Index > fa.nextIssueIndex {
			fa.nextIssueIndex = issue.Index
		}
	}
	for _, comment := range fa.Comments {
		if comment.ID > fa.nextCommentID {
			fa.nextCommentID = comment.ID
		}
	}
	for _, comments := range fa.CommentsByIssue {
		for _, comment := range comments {
			if comment.ID > fa.nextCommentID {
				fa.nextCommentID = comment.ID
			}
		}
	}
	for _, label := range append(append([]*gitea.Label{}, fa.RepoLabels...), fa.Labels...) {
		if label.ID > fa.nextLabelID {
			fa.nextLabelID = label.ID
		}
	}
	for _, labels := range fa.LabelsByIssue {
		for _, label := range labels {
			if label.ID > fa.nextLabelID {
				fa.nextLabelID = label.ID
			}
		}
	}
}

func (fa *FakeAPI) findIssue(index int64) (*gitea.Issue, bool) {
	for _, issue := range fa.Issues {
		if issue.Index == index {
			return issue, true
		}
	}
	return nil, false
}

func (fa *FakeAPI) findRepoLabel(id int64) (*gitea.Label, bool) {
	for _, label := range fa.RepoLabels {
		if label.ID == id {
			return label, true
		}
	}
	return nil, false
}

func (fa *FakeAPI) issueLabelsFromIDs(ids []int64) []*gitea.Label {
	labels := make([]*gitea.Label, 0, len(ids))
	for _, id := range ids {
		if label, ok := fa.findRepoLabel(id); ok {
			labels = append(labels, label)
		}
	}
	return labels
}

func (fa *FakeAPI) setIssueLabels(index int64, labels []*gitea.Label) {
	if fa.LabelsByIssue == nil {
		fa.LabelsByIssue = map[int64][]*gitea.Label{}
	}
	fa.LabelsByIssue[index] = labels
	if index == 1 {
		fa.Labels = labels
	}
	if issue, ok := fa.findIssue(index); ok {
		issue.Labels = labels
	}
}

func (fa *FakeAPI) appendIssueLabels(index int64, labels []*gitea.Label) []*gitea.Label {
	current := append([]*gitea.Label(nil), fa.labelsFor(index)...)
	seen := map[int64]bool{}
	for _, label := range current {
		seen[label.ID] = true
	}
	for _, label := range labels {
		if !seen[label.ID] {
			current = append(current, label)
			seen[label.ID] = true
		}
	}
	fa.setIssueLabels(index, current)
	return current
}

func (fa *FakeAPI) removeIssueLabel(index int64, identifier string) bool {
	id, idErr := strconv.ParseInt(identifier, 10, 64)
	labels := fa.labelsFor(index)
	filtered := labels[:0]
	removed := false
	for _, label := range labels {
		match := label.Name == identifier
		if idErr == nil {
			match = label.ID == id
		}
		if match {
			removed = true
			continue
		}
		filtered = append(filtered, label)
	}
	fa.setIssueLabels(index, filtered)
	return removed
}

// NewServer starts an httptest.Server backed by this FakeAPI and registers
// a cleanup to close it when t finishes.
func (fa *FakeAPI) NewServer(t *testing.T) *httptest.Server {
	t.Helper()
	fa.initSequences()
	mux := http.NewServeMux()

	fa.registerIssueHandlers(mux)
	fa.registerLabelHandlers(mux)
	fa.registerMiscHandlers(mux)

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func (fa *FakeAPI) registerIssueHandlers(mux *http.ServeMux) {
	issuesPath := fmt.Sprintf("/api/v1/repos/%s/%s/issues", fa.Owner, fa.Project)
	issuesPrefix := issuesPath + "/"
	mux.HandleFunc(issuesPath, fa.handleIssues)
	mux.HandleFunc(issuesPrefix, fa.handleIssueSubresource(issuesPrefix))
}

func (fa *FakeAPI) registerLabelHandlers(mux *http.ServeMux) {
	labelsPath := fmt.Sprintf("/api/v1/repos/%s/%s/labels", fa.Owner, fa.Project)
	labelsPrefix := labelsPath + "/"
	mux.HandleFunc(labelsPath, fa.handleRepoLabels(labelsPath))
	mux.HandleFunc(labelsPrefix, fa.handleRepoLabel(labelsPrefix))
}

func (fa *FakeAPI) handleIssues(w http.ResponseWriter, r *http.Request) {
	fa.IssueRequests = append(fa.IssueRequests, r)
	switch r.Method {
	case http.MethodPost:
		fa.createIssue(w, r)
	case http.MethodGet:
		fa.listIssues(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (fa *FakeAPI) createIssue(w http.ResponseWriter, r *http.Request) {
	opt, ok := decodeJSON[gitea.CreateIssueOption](w, r)
	if !ok {
		return
	}
	fa.nextIssueID++
	fa.nextIssueIndex++
	now := time.Now().UTC()
	state := gitea.StateOpen
	var closed *time.Time
	if opt.Closed {
		state = gitea.StateClosed
		closed = &now
	}
	issue := &gitea.Issue{
		ID:      fa.nextIssueID,
		Index:   fa.nextIssueIndex,
		Title:   opt.Title,
		Body:    opt.Body,
		State:   state,
		Poster:  &gitea.User{UserName: "testuser"},
		Labels:  fa.issueLabelsFromIDs(opt.Labels),
		Created: now,
		Updated: now,
		Closed:  closed,
	}
	fa.Issues = append(fa.Issues, issue)
	fa.setIssueLabels(issue.Index, issue.Labels)
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, issue)
}

func (fa *FakeAPI) listIssues(w http.ResponseWriter, r *http.Request) {
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
}

func (fa *FakeAPI) handleIssueSubresource(issuesPrefix string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rest := strings.TrimPrefix(r.URL.Path, issuesPrefix)
		if strings.HasPrefix(rest, "comments/") {
			fa.handleIssueCommentByID(w, r, strings.TrimPrefix(rest, "comments/"))
			return
		}

		parts := strings.SplitN(rest, "/", 2)
		idx, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		if len(parts) < 2 {
			fa.handleIssueByIndex(w, r, idx)
			return
		}

		switch parts[1] {
		case "comments":
			fa.handleIssueComments(w, r, idx)
		case "labels":
			fa.handleIssueLabels(w, r, idx)
		default:
			if strings.HasPrefix(parts[1], "labels/") && r.Method == http.MethodDelete {
				identifier := strings.TrimPrefix(parts[1], "labels/")
				if fa.removeIssueLabel(idx, identifier) {
					w.WriteHeader(http.StatusNoContent)
					return
				}
			}
			http.NotFound(w, r)
		}
	}
}

func (fa *FakeAPI) handleIssueCommentByID(w http.ResponseWriter, r *http.Request, rawID string) {
	commentID, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodPatch {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	opt, ok := decodeJSON[gitea.EditIssueCommentOption](w, r)
	if !ok {
		return
	}
	comment, ok := fa.findComment(commentID)
	if !ok {
		http.NotFound(w, r)
		return
	}
	comment.Body = opt.Body
	comment.Updated = time.Now().UTC()
	writeJSON(w, comment)
}

func (fa *FakeAPI) findComment(id int64) (*gitea.Comment, bool) {
	for _, comments := range fa.CommentsByIssue {
		for _, comment := range comments {
			if comment.ID == id {
				return comment, true
			}
		}
	}
	for _, comment := range fa.Comments {
		if comment.ID == id {
			return comment, true
		}
	}
	return nil, false
}

func (fa *FakeAPI) handleIssueByIndex(w http.ResponseWriter, r *http.Request, idx int64) {
	if r.Method != http.MethodPatch {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	issue, ok := fa.findIssue(idx)
	if !ok {
		http.NotFound(w, r)
		return
	}
	opt, ok := decodeJSON[gitea.EditIssueOption](w, r)
	if !ok {
		return
	}
	if opt.Title != "" {
		issue.Title = opt.Title
	}
	if opt.Body != nil {
		issue.Body = *opt.Body
	}
	if opt.State != nil {
		issue.State = *opt.State
		if *opt.State == gitea.StateClosed {
			now := time.Now().UTC()
			issue.Closed = &now
		} else {
			issue.Closed = nil
		}
	}
	issue.Updated = time.Now().UTC()
	writeJSON(w, issue)
}

func (fa *FakeAPI) handleIssueComments(w http.ResponseWriter, r *http.Request, idx int64) {
	switch r.Method {
	case http.MethodPost:
		fa.createIssueComment(w, r, idx)
	case http.MethodGet:
		fa.listIssueComments(w, r, idx)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (fa *FakeAPI) createIssueComment(w http.ResponseWriter, r *http.Request, idx int64) {
	opt, ok := decodeJSON[gitea.CreateIssueCommentOption](w, r)
	if !ok {
		return
	}
	fa.nextCommentID++
	now := time.Now().UTC()
	comment := &gitea.Comment{
		ID:      fa.nextCommentID,
		Body:    opt.Body,
		Poster:  &gitea.User{UserName: "testuser"},
		Created: now,
		Updated: now,
	}
	if fa.CommentsByIssue == nil {
		fa.CommentsByIssue = map[int64][]*gitea.Comment{}
	}
	fa.CommentsByIssue[idx] = append(fa.commentsFor(idx), comment)
	if idx == 1 {
		fa.Comments = fa.CommentsByIssue[idx]
	}
	writeJSON(w, comment)
}

func (fa *FakeAPI) listIssueComments(w http.ResponseWriter, r *http.Request, idx int64) {
	fa.CommentRequests = append(fa.CommentRequests, r)
	page, limit := parsePagination(r)
	if fa.CommentErrPage > 0 && page == fa.CommentErrPage {
		http.Error(w, "simulated server error", http.StatusInternalServerError)
		return
	}
	comments := fa.commentsFor(idx)
	w.Header().Set("X-Total-Count", strconv.Itoa(len(comments)))
	start, end := pageSlice(page, limit, len(comments))
	writeJSON(w, comments[start:end])
}

func (fa *FakeAPI) handleIssueLabels(w http.ResponseWriter, r *http.Request, idx int64) {
	switch r.Method {
	case http.MethodPost:
		fa.addIssueLabels(w, r, idx)
	case http.MethodPut:
		fa.replaceIssueLabels(w, r, idx)
	case http.MethodDelete:
		fa.setIssueLabels(idx, nil)
		w.WriteHeader(http.StatusNoContent)
	case http.MethodGet:
		fa.listIssueLabels(w, r, idx)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (fa *FakeAPI) addIssueLabels(w http.ResponseWriter, r *http.Request, idx int64) {
	opt, ok := decodeJSON[gitea.IssueLabelsOption](w, r)
	if !ok {
		return
	}
	writeJSON(w, fa.appendIssueLabels(idx, fa.issueLabelsFromIDs(opt.Labels)))
}

func (fa *FakeAPI) replaceIssueLabels(w http.ResponseWriter, r *http.Request, idx int64) {
	opt, ok := decodeJSON[gitea.IssueLabelsOption](w, r)
	if !ok {
		return
	}
	labels := fa.issueLabelsFromIDs(opt.Labels)
	fa.setIssueLabels(idx, labels)
	writeJSON(w, labels)
}

func (fa *FakeAPI) listIssueLabels(w http.ResponseWriter, r *http.Request, idx int64) {
	page, limit := parsePagination(r)
	if fa.LabelErrPage > 0 && page == fa.LabelErrPage {
		http.Error(w, "simulated server error", http.StatusInternalServerError)
		return
	}
	labels := fa.labelsFor(idx)
	start, end := pageSlice(page, limit, len(labels))
	writeJSON(w, labels[start:end])
}

func (fa *FakeAPI) handleRepoLabels(labelsPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			page, limit := parsePagination(r)
			start, end := pageSlice(page, limit, len(fa.RepoLabels))
			writeJSON(w, fa.RepoLabels[start:end])
		case http.MethodPost:
			fa.createRepoLabel(w, r, labelsPath)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func (fa *FakeAPI) createRepoLabel(w http.ResponseWriter, r *http.Request, labelsPath string) {
	opt, ok := decodeJSON[gitea.CreateLabelOption](w, r)
	if !ok {
		return
	}
	fa.nextLabelID++
	label := &gitea.Label{
		ID:          fa.nextLabelID,
		Name:        opt.Name,
		Color:       opt.Color,
		Description: opt.Description,
		Exclusive:   opt.Exclusive,
		IsArchived:  opt.IsArchived,
		URL:         fmt.Sprintf("%s/%d", labelsPath, fa.nextLabelID),
	}
	fa.RepoLabels = append(fa.RepoLabels, label)
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, label)
}

func (fa *FakeAPI) handleRepoLabel(labelsPrefix string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(strings.TrimPrefix(r.URL.Path, labelsPrefix), 10, 64)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		label, ok := fa.findRepoLabel(id)
		if !ok {
			http.NotFound(w, r)
			return
		}
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, label)
		case http.MethodPatch:
			fa.editRepoLabel(w, r, label)
		case http.MethodDelete:
			fa.deleteRepoLabel(id)
			w.WriteHeader(http.StatusNoContent)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func (fa *FakeAPI) editRepoLabel(w http.ResponseWriter, r *http.Request, label *gitea.Label) {
	opt, ok := decodeJSON[gitea.EditLabelOption](w, r)
	if !ok {
		return
	}
	if opt.Name != nil {
		label.Name = *opt.Name
	}
	if opt.Color != nil {
		label.Color = *opt.Color
	}
	if opt.Description != nil {
		label.Description = *opt.Description
	}
	if opt.Exclusive != nil {
		label.Exclusive = *opt.Exclusive
	}
	if opt.IsArchived != nil {
		label.IsArchived = *opt.IsArchived
	}
	writeJSON(w, label)
}

func (fa *FakeAPI) deleteRepoLabel(id int64) {
	filtered := fa.RepoLabels[:0]
	for _, existing := range fa.RepoLabels {
		if existing.ID != id {
			filtered = append(filtered, existing)
		}
	}
	fa.RepoLabels = filtered
}

func (fa *FakeAPI) registerMiscHandlers(mux *http.ServeMux) {
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
		if fa.RepoErrStatus != 0 {
			http.Error(w, http.StatusText(fa.RepoErrStatus), fa.RepoErrStatus)
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
		fa.UserRequests = append(fa.UserRequests, r)
		login := strings.TrimPrefix(r.URL.Path, "/api/v1/users/")
		if fa.UserNetworkErrLogin != "" && login == fa.UserNetworkErrLogin {
			// Abort the response without writing anything; the client sees
			// an EOF and returns (nil, err) — no *Response to inspect.
			panic(http.ErrAbortHandler)
		}
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

}
