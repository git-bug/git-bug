package http

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/git-bug/git-bug/bridge"
	"github.com/git-bug/git-bug/bridge/core"
	"github.com/git-bug/git-bug/cache"
)

// SyncHandler exposes a two-method endpoint (GET/POST) that runs
// `git-bug bridge pull` across every registered repository in parallel.
//
//   POST /sync  — start a sync run. Returns 409 if one is already in flight.
//   GET  /sync  — return the current status as JSON.
//
// Sync is a background goroutine pool; the HTTP response returns
// immediately. The UI polls GET until Running becomes false.
type SyncHandler struct {
	mrc *cache.MultiRepoCache

	mu     sync.Mutex
	status SyncStatus
	cancel context.CancelFunc
}

// SyncStatus is the shape returned by GET /sync and is also the inline
// state held on SyncHandler. It is intentionally flat and JSON-friendly.
type SyncStatus struct {
	Running    bool      `json:"running"`
	StartedAt  time.Time `json:"startedAt,omitempty"`
	FinishedAt time.Time `json:"finishedAt,omitempty"`
	// Active is the set of repo names currently being synced. Up to
	// syncConcurrency entries at a time.
	Active []string          `json:"active"`
	Done   int               `json:"done"`
	Total  int               `json:"total"`
	Errors map[string]string `json:"errors,omitempty"`
	// Cumulative counters for the most recent run.
	ImportedBugs       int `json:"importedBugs"`
	ImportedIdentities int `json:"importedIdentities"`
}

// syncConcurrency caps parallel bridge pulls. Higher values speed up large
// trees but burn GitHub's 5000/hr GraphQL rate limit quickly; 3 matches
// the shell script's MAX_JOBS default and has proven safe in practice.
const syncConcurrency = 3

func NewSyncHandler(mrc *cache.MultiRepoCache) *SyncHandler {
	return &SyncHandler{
		mrc:    mrc,
		status: SyncStatus{Active: []string{}, Errors: map[string]string{}},
	}
}

func (h *SyncHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.writeStatus(w)
	case http.MethodPost:
		h.start(w)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *SyncHandler) writeStatus(w http.ResponseWriter) {
	h.mu.Lock()
	snap := h.status
	// Defensive copies so callers don't see concurrent mutation.
	errs := make(map[string]string, len(h.status.Errors))
	for k, v := range h.status.Errors {
		errs[k] = v
	}
	snap.Errors = errs
	active := make([]string, len(h.status.Active))
	copy(active, h.status.Active)
	snap.Active = active
	h.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(snap)
}

func (h *SyncHandler) start(w http.ResponseWriter) {
	h.mu.Lock()
	if h.status.Running {
		h.mu.Unlock()
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "sync already running"})
		return
	}
	repos := h.mrc.AllRepos()
	ctx, cancel := context.WithCancel(context.Background())
	h.cancel = cancel
	h.status = SyncStatus{
		Running:   true,
		StartedAt: time.Now(),
		Total:     len(repos),
		Active:    []string{},
		Errors:    map[string]string{},
	}
	h.mu.Unlock()

	go h.run(ctx, repos)

	w.WriteHeader(http.StatusAccepted)
	h.writeStatus(w)
}

func (h *SyncHandler) run(ctx context.Context, repos []*cache.RepoCache) {
	// Deterministic order by name so the UI shows a predictable progression.
	sort.Slice(repos, func(i, j int) bool { return repos[i].Name() < repos[j].Name() })

	// Dispatch through a buffered channel acting as a semaphore; workers
	// drain it and each handles one repo at a time.
	jobs := make(chan *cache.RepoCache)
	var wg sync.WaitGroup
	for i := 0; i < syncConcurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for repo := range jobs {
				if ctx.Err() != nil {
					return
				}
				h.markActive(repo.Name(), true)
				bugs, ids, err := syncOneRepo(ctx, repo)
				h.markActive(repo.Name(), false)

				h.mu.Lock()
				if err != nil {
					h.status.Errors[repo.Name()] = err.Error()
				}
				h.status.Done++
				h.status.ImportedBugs += bugs
				h.status.ImportedIdentities += ids
				h.mu.Unlock()
			}
		}()
	}

	for _, repo := range repos {
		select {
		case jobs <- repo:
		case <-ctx.Done():
			close(jobs)
			wg.Wait()
			h.finish()
			return
		}
	}
	close(jobs)
	wg.Wait()
	h.finish()
}

func (h *SyncHandler) markActive(name string, entering bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if entering {
		h.status.Active = append(h.status.Active, name)
		return
	}
	out := h.status.Active[:0]
	for _, n := range h.status.Active {
		if n != name {
			out = append(out, n)
		}
	}
	h.status.Active = out
}

func (h *SyncHandler) finish() {
	h.mu.Lock()
	h.status.Running = false
	h.status.FinishedAt = time.Now()
	h.status.Active = []string{}
	h.mu.Unlock()
}

// syncOneRepo runs the equivalent of `git-bug bridge pull` on a single repo.
// Repos without any bridge configured are treated as no-ops, not errors.
func syncOneRepo(ctx context.Context, repo *cache.RepoCache) (int, int, error) {
	b, err := bridge.DefaultBridge(repo)
	if err != nil {
		return 0, 0, nil
	}

	events, err := b.ImportAll(ctx)
	if err != nil {
		return 0, 0, err
	}

	bugs := 0
	ids := 0
	for result := range events {
		switch result.Event {
		case core.ImportEventBug:
			bugs++
		case core.ImportEventIdentity:
			ids++
		case core.ImportEventError:
			if result.Err != context.Canceled {
				return bugs, ids, result.Err
			}
		}
	}
	return bugs, ids, nil
}
