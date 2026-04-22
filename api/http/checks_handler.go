package http

import (
	"context"
	"encoding/json"
	"net/http"
	"regexp"
	"sync"
	"time"

	"github.com/gorilla/mux"

	"github.com/git-bug/git-bug/bridge/github"
	"github.com/git-bug/git-bug/cache"
)

// ChecksHandler serves GET /checks/{repo}/{sha} — returns the current
// GitHub check-suite state for a commit on the given repo. Responses are
// cached per-key for checksTTL so a busy PR page doesn't hammer GitHub's
// rate limit; every GitHub call costs ~1 GraphQL point and refreshes the
// whole (rollup + suites + runs) tree.
type ChecksHandler struct {
	mrc *cache.MultiRepoCache

	mu    sync.Mutex
	cache map[string]cachedChecks
}

type cachedChecks struct {
	at  time.Time
	val *github.CheckStatus
	err string // empty on success; the error message is surfaced as-is
}

// checksTTL is the freshness window for cached responses. CI state moves on
// the order of seconds for in-progress runs, but refetching on every tab
// flip is pure waste. 30s is the sweet spot observed on GitHub's own UI.
const checksTTL = 30 * time.Second

// shaRe keeps unexpected paths from hitting GitHub. The 40/64 bounds match
// git SHA-1 and SHA-256 lengths.
var shaRe = regexp.MustCompile(`^[0-9a-f]{40}([0-9a-f]{24})?$`)

func NewChecksHandler(mrc *cache.MultiRepoCache) *ChecksHandler {
	return &ChecksHandler{
		mrc:   mrc,
		cache: map[string]cachedChecks{},
	}
}

func (h *ChecksHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	vars := mux.Vars(r)
	repoName := vars["repo"]
	sha := vars["sha"]

	if !shaRe.MatchString(sha) {
		http.Error(w, "invalid commit sha", http.StatusBadRequest)
		return
	}

	repo, err := h.mrc.ResolveRepo(repoName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	key := repoName + "@" + sha
	if cached, ok := h.lookup(key); ok {
		writeCheckResponse(w, cached.val, cached.err, true)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	status, fetchErr := github.FetchCheckStatus(ctx, repo, sha)

	errMsg := ""
	if fetchErr != nil {
		errMsg = fetchErr.Error()
	}
	h.store(key, cachedChecks{at: time.Now(), val: status, err: errMsg})
	writeCheckResponse(w, status, errMsg, false)
}

func (h *ChecksHandler) lookup(key string) (cachedChecks, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	c, ok := h.cache[key]
	if !ok || time.Since(c.at) > checksTTL {
		return cachedChecks{}, false
	}
	return c, true
}

func (h *ChecksHandler) store(key string, c cachedChecks) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.cache[key] = c
	// Cheap GC: if the map has grown past a sensible bound, drop stale
	// entries. The bound (2048) is generous for a single-user dev tool.
	if len(h.cache) > 2048 {
		cutoff := time.Now().Add(-2 * checksTTL)
		for k, v := range h.cache {
			if v.at.Before(cutoff) {
				delete(h.cache, k)
			}
		}
	}
}

// checkResponse is the HTTP response shape. We always include the error
// field (possibly empty) so the UI has a single code path for both success
// and failure.
type checkResponse struct {
	*github.CheckStatus
	// Source is always "github.com" for now — explicit label so the UI
	// can render "CI: green (source: github.com, requires trust)" as
	// designed in the distributed-CI discussion.
	Source string `json:"source"`
	Error  string `json:"error,omitempty"`
	Cached bool   `json:"cached"`
}

func writeCheckResponse(w http.ResponseWriter, status *github.CheckStatus, errMsg string, cached bool) {
	w.Header().Set("Content-Type", "application/json")
	if errMsg != "" && status == nil {
		w.WriteHeader(http.StatusBadGateway)
	}
	_ = json.NewEncoder(w).Encode(checkResponse{
		CheckStatus: status,
		Source:      "github.com",
		Error:       errMsg,
		Cached:      cached,
	})
}
