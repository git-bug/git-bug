package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"

	"github.com/git-bug/git-bug/cache"
	"github.com/git-bug/git-bug/repository"
)

// PRHandler serves read-only views of a pull-request's commits and the
// aggregate diff between its base and head. Data comes from the local git
// repo (via repository.RepoBrowse) — it's free to compute and doesn't
// touch GitHub at all.
//
// Routes:
//
//	GET /pr/{repo}/commits?base=<ref>&head=<ref>&limit=<n>
//	GET /pr/{repo}/diff?base=<ref>&head=<ref>
type PRHandler struct {
	mrc *cache.MultiRepoCache

	// pullFetchMu guards pullFetchAt. Fetches for refs/pull/<n>/head
	// round-trip to GitHub, so we skip repeat fetches within pullFetchTTL
	// to keep tab-flipping snappy while still picking up new pushes on a
	// reasonable cadence.
	pullFetchMu sync.Mutex
	pullFetchAt map[string]time.Time
}

// pullFetchTTL bounds how often we re-fetch a given PR ref. 60 s keeps
// "user is actively viewing this PR" refreshes reasonable without
// spamming GitHub when a user clicks between Commits / Files / Checks
// tabs.
const pullFetchTTL = 60 * time.Second

func NewPRHandler(mrc *cache.MultiRepoCache) *PRHandler {
	return &PRHandler{
		mrc:         mrc,
		pullFetchAt: map[string]time.Time{},
	}
}

func (h *PRHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	vars := mux.Vars(r)

	repoName := vars["repo"]
	repo, err := h.mrc.ResolveRepo(repoName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	q := r.URL.Query()
	base := strings.TrimSpace(q.Get("base"))
	head := strings.TrimSpace(q.Get("head"))
	if base == "" || head == "" {
		http.Error(w, "missing ?base or ?head", http.StatusBadRequest)
		return
	}

	// Parse the optional PR number. We'll use it as a fallback — if the
	// head the caller passed is already local, we skip the fetch and
	// the PR number isn't needed.
	pr := 0
	if prStr := q.Get("pr"); prStr != "" {
		if n, err := strconv.Atoi(prStr); err == nil && n > 0 {
			pr = n
		}
	}

	// Keep the route shape consistent with /checks/ and /sync — use the
	// last path element as the action discriminator.
	switch vars["action"] {
	case "commits":
		h.serveCommits(w, repoName, repo, base, head, pr, q.Get("limit"))
	case "diff":
		h.serveDiff(w, repoName, repo, base, head, pr)
	default:
		http.Error(w, "unknown PR action", http.StatusNotFound)
	}
}

// resolveOrFetch returns the effective head ref to use — preferring the
// caller-supplied head when it's already in our local object store, and
// only fetching refs/pull/<n>/head on-demand when the commit is missing.
//
// This is the key to cheap tab-flipping: once a repo has been synced
// (git-bug recorded the PR head, and the sync pulled the fork's commit),
// every subsequent request is a local resolve — no shell-out, no network.
// The fetch fallback only kicks in for fork PRs we've never seen, or
// when a force-push has invalidated the stored SHA.
func (h *PRHandler) resolveOrFetch(repoKey string, repo *cache.RepoCache, head string, pr int) (string, error) {
	if headIsLocal(repo, head) {
		return head, nil
	}
	if pr > 0 && h.ensurePullRef(repoKey, repo, pr) {
		return fmt.Sprintf("refs/remotes/origin/pr/%d", pr), nil
	}
	return "", repository.ErrNotFound
}

// headIsLocal reports whether the caller-supplied head ref/SHA already
// resolves in the local repository. Saves a network round-trip per page
// load once the PR has been fetched at least once.
//
// Probe via CommitsAhead(head, head, 1) — that calls the gogit
// resolveRefToHash on both arguments (which covers raw hash and the
// refs/heads → origin/<ref> fallback), then walks zero commits. Returns
// ErrNotFound if either ref is missing; nil on hit.
func headIsLocal(repo *cache.RepoCache, head string) bool {
	if head == "" {
		return false
	}
	_, err := repo.BrowseRepo().CommitsAhead(head, head, 1)
	return err == nil
}

// ensurePullRef fetches refs/pull/<n>/head so the head commit of a fork
// PR lands in the local repo, then caches the timestamp so we don't
// re-fetch within pullFetchTTL. Idempotent: the second call within the
// window is a map lookup, no git invocation, no network. Returns true
// when origin/pr/<n> should be usable; false on fetch errors.
func (h *PRHandler) ensurePullRef(repoKey string, repo *cache.RepoCache, n int) bool {
	key := fmt.Sprintf("%s#%d", repoKey, n)

	h.pullFetchMu.Lock()
	last, ok := h.pullFetchAt[key]
	h.pullFetchMu.Unlock()
	if ok && time.Since(last) < pullFetchTTL {
		return true
	}

	spec := fmt.Sprintf("refs/pull/%d/head:refs/remotes/origin/pr/%d", n, n)
	if _, err := repo.FetchRefSpecs("origin", []string{spec}); err != nil {
		return false
	}

	h.pullFetchMu.Lock()
	h.pullFetchAt[key] = time.Now()
	h.pullFetchMu.Unlock()
	return true
}

// commitsResponse is the wire format for the Commits tab. Ordered newest
// first — the UI shows them reversed (old-at-top, new-at-bottom) which
// matches how commits appear in a PR.
type commitsResponse struct {
	Commits []commitEntry `json:"commits"`
}

type commitEntry struct {
	Hash        string `json:"hash"`
	Message     string `json:"message"`
	AuthorName  string `json:"authorName"`
	AuthorEmail string `json:"authorEmail"`
	Date        string `json:"date"`
}

func (h *PRHandler) serveCommits(w http.ResponseWriter, repoName string, repo *cache.RepoCache, base, head string, pr int, limitStr string) {
	limit := 200
	if limitStr != "" {
		if n, err := strconv.Atoi(limitStr); err == nil && n > 0 {
			limit = n
		}
	}
	effHead, err := h.resolveOrFetch(repoName, repo, head, pr)
	if err != nil {
		http.Error(w, refNotFoundHint(base, head), http.StatusNotFound)
		return
	}
	commits, err := repo.BrowseRepo().CommitsAhead(base, effHead, limit)
	if err != nil {
		httpStatus := http.StatusInternalServerError
		msg := err.Error()
		if err == repository.ErrNotFound {
			httpStatus = http.StatusNotFound
			msg = refNotFoundHint(base, effHead)
		}
		http.Error(w, msg, httpStatus)
		return
	}
	out := commitsResponse{Commits: make([]commitEntry, 0, len(commits))}
	for _, c := range commits {
		out.Commits = append(out.Commits, commitEntry{
			Hash:        string(c.Hash),
			Message:     c.Message,
			AuthorName:  c.AuthorName,
			AuthorEmail: c.AuthorEmail,
			Date:        c.Date.Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

// diffResponse is the wire format for the Files Changed tab.
type diffResponse struct {
	Files []fileDiffEntry `json:"files"`
}

type fileDiffEntry struct {
	Path     string         `json:"path"`
	OldPath  string         `json:"oldPath,omitempty"`
	IsNew    bool           `json:"isNew,omitempty"`
	IsDelete bool           `json:"isDelete,omitempty"`
	IsBinary bool           `json:"isBinary,omitempty"`
	Additions int           `json:"additions"`
	Deletions int           `json:"deletions"`
	Hunks    []diffHunkEntry `json:"hunks,omitempty"`
}

type diffHunkEntry struct {
	Header   string          `json:"header"`
	OldStart int             `json:"oldStart"`
	OldLines int             `json:"oldLines"`
	NewStart int             `json:"newStart"`
	NewLines int             `json:"newLines"`
	Lines    []diffLineEntry `json:"lines"`
}

type diffLineEntry struct {
	Type    string `json:"type"`    // "add", "del", "ctx"
	Old     int    `json:"old,omitempty"`
	New     int    `json:"new,omitempty"`
	Content string `json:"content"`
}

func (h *PRHandler) serveDiff(w http.ResponseWriter, repoName string, repo *cache.RepoCache, base, head string, pr int) {
	effHead, err := h.resolveOrFetch(repoName, repo, head, pr)
	if err != nil {
		http.Error(w, refNotFoundHint(base, head), http.StatusNotFound)
		return
	}
	files, err := repo.BrowseRepo().DiffBetween(base, effHead)
	if err != nil {
		httpStatus := http.StatusInternalServerError
		msg := err.Error()
		if err == repository.ErrNotFound {
			httpStatus = http.StatusNotFound
			msg = refNotFoundHint(base, effHead)
		}
		http.Error(w, msg, httpStatus)
		return
	}

	out := diffResponse{Files: make([]fileDiffEntry, 0, len(files))}
	for _, f := range files {
		entry := fileDiffEntry{
			Path:     f.Path,
			IsNew:    f.IsNew,
			IsDelete: f.IsDelete,
			IsBinary: f.IsBinary,
		}
		if f.OldPath != nil {
			entry.OldPath = *f.OldPath
		}
		for _, h := range f.Hunks {
			he := diffHunkEntry{
				Header:   hunkHeader(h),
				OldStart: h.OldStart,
				OldLines: h.OldLines,
				NewStart: h.NewStart,
				NewLines: h.NewLines,
			}
			for _, l := range h.Lines {
				he.Lines = append(he.Lines, diffLineEntry{
					Type:    lineType(l.Type),
					Old:     l.OldLine,
					New:     l.NewLine,
					Content: l.Content,
				})
				switch l.Type {
				case repository.DiffLineAdded:
					entry.Additions++
				case repository.DiffLineDeleted:
					entry.Deletions++
				}
			}
			entry.Hunks = append(entry.Hunks, he)
		}
		out.Files = append(out.Files, entry)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

// hunkHeader assembles a @@ header a la `git diff` for display.
func hunkHeader(h repository.DiffHunk) string {
	return "@@ -" + itoa(h.OldStart) + "," + itoa(h.OldLines) + " +" +
		itoa(h.NewStart) + "," + itoa(h.NewLines) + " @@"
}

func itoa(n int) string { return strconv.Itoa(n) }

// refNotFoundHint crafts a user-facing message that names which ref was
// the problem — much more actionable than a bare "ref not found".
func refNotFoundHint(base, head string) string {
	return "git ref or commit not fetched locally: tried base=" + base +
		", head=" + head +
		". If this is a fork PR, the head commit may not be in the mirror; " +
		"add `refs/pull/*/head:refs/remotes/origin/pr/*` to the fetch refspec " +
		"and re-fetch to populate it."
}

func lineType(t repository.DiffLineType) string {
	switch t {
	case repository.DiffLineAdded:
		return "add"
	case repository.DiffLineDeleted:
		return "del"
	default:
		return "ctx"
	}
}
