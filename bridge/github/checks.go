package github

import (
	"context"
	"fmt"
	"time"

	"github.com/shurcooL/githubv4"

	"github.com/git-bug/git-bug/bridge/core/auth"
	"github.com/git-bug/git-bug/cache"
)

// CheckStatus is the flattened, UI-ready shape returned by FetchCheckStatus.
// We hoist GitHub's nested connection-of-connections into a shallow list so
// the webui can render it without re-walking the GraphQL response.
type CheckStatus struct {
	// State is the aggregate rollup. One of
	// "SUCCESS" | "PENDING" | "FAILURE" | "ERROR" | "EXPECTED" | "" (unknown).
	State string       `json:"state"`
	Suites []CheckSuite `json:"suites"`
	// FetchedAt lets the UI show "data was fresh N seconds ago" and helps
	// callers tell cached responses from live ones.
	FetchedAt time.Time `json:"fetchedAt"`
}

// CheckSuite corresponds to one workflow/app's set of checks on the commit.
// The app distinguishes e.g. "GitHub Actions" from a third-party CI bot.
type CheckSuite struct {
	AppName    string     `json:"appName"`
	Status     string     `json:"status"`     // IN_PROGRESS | QUEUED | COMPLETED | …
	Conclusion string     `json:"conclusion"` // SUCCESS | FAILURE | CANCELLED | SKIPPED | …
	Runs       []CheckRun `json:"runs"`
}

// CheckRun is an individual job inside a suite (e.g. a matrix entry).
type CheckRun struct {
	Name       string `json:"name"`
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
	DetailsUrl string `json:"detailsUrl,omitempty"`
	// StartedAt and CompletedAt are RFC3339; blank if unavailable.
	StartedAt   string `json:"startedAt,omitempty"`
	CompletedAt string `json:"completedAt,omitempty"`
}

// checkStatusQuery is the raw GraphQL shape. Kept internal — callers consume
// CheckStatus, which is stable across GitHub's schema quirks.
//
// Cost: both inner connections are first:<=100, so the total rate-limit cost
// is roughly 1 point per fetch. We cache responses in the handler to avoid
// paying for repeated page loads.
type checkStatusQuery struct {
	Repository struct {
		Object struct {
			Typename githubv4.String `graphql:"__typename"`
			Commit   struct {
				StatusCheckRollup *struct {
					State githubv4.String
				}
				CheckSuites struct {
					Nodes []struct {
						App *struct {
							Name githubv4.String
						}
						Status     githubv4.String
						Conclusion githubv4.String
						CheckRuns  struct {
							Nodes []struct {
								Name        githubv4.String
								Status      githubv4.String
								Conclusion  githubv4.String
								DetailsUrl  githubv4.URI
								StartedAt   githubv4.DateTime
								CompletedAt githubv4.DateTime
							}
						} `graphql:"checkRuns(first: 30)"`
					}
				} `graphql:"checkSuites(first: 20)"`
			} `graphql:"... on Commit"`
		} `graphql:"object(oid: $oid)"`
	} `graphql:"repository(owner: $owner, name: $name)"`
	RateLimit rateLimit
}

// FetchCheckStatus queries GitHub for the current check state of a commit on
// a repo's configured default bridge. Returns a nil *CheckStatus and no error
// when the repo has no github bridge configured — CI status is an optional
// enhancement, not a required field.
//
// No caching here; callers (the HTTP handler) wrap this with a TTL cache so
// a busy PR page doesn't thrash the rate limit.
func FetchCheckStatus(ctx context.Context, repo *cache.RepoCache, sha string) (*CheckStatus, error) {
	if sha == "" {
		return nil, fmt.Errorf("commit sha is required")
	}

	// The commit-attached conf lives alongside the bridge config under
	// git-bug.bridge.github.*. We look up owner/project there rather than
	// parsing from a remote URL — the bridge already did that work.
	cfg, err := readGithubConfig(repo)
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		return nil, nil // no github bridge — no check status
	}

	creds, err := auth.List(repo,
		auth.WithTarget(target),
		auth.WithKind(auth.KindToken),
		auth.WithMeta(auth.MetaKeyLogin, cfg.login),
	)
	if err != nil {
		return nil, fmt.Errorf("find github token: %w", err)
	}
	if len(creds) == 0 {
		return nil, ErrMissingIdentityToken
	}

	client := buildClient(creds[0].(*auth.Token))

	var q checkStatusQuery
	vars := map[string]interface{}{
		"owner": githubv4.String(cfg.owner),
		"name":  githubv4.String(cfg.project),
		"oid":   githubv4.GitObjectID(sha),
	}
	if err := client.sc.Query(ctx, &q, vars); err != nil {
		return nil, fmt.Errorf("github checkStatus query: %w", err)
	}
	noteRateLimit("checkStatus", cfg.owner, cfg.project, q.RateLimit)

	// __typename must be "Commit" — if the SHA doesn't resolve to one the
	// Object block is empty and State/Suites stay at zero-values.
	if q.Repository.Object.Typename != "Commit" {
		return &CheckStatus{FetchedAt: time.Now()}, nil
	}

	out := &CheckStatus{FetchedAt: time.Now()}
	if rollup := q.Repository.Object.Commit.StatusCheckRollup; rollup != nil {
		out.State = string(rollup.State)
	}
	for _, s := range q.Repository.Object.Commit.CheckSuites.Nodes {
		suite := CheckSuite{
			Status:     string(s.Status),
			Conclusion: string(s.Conclusion),
		}
		if s.App != nil {
			suite.AppName = string(s.App.Name)
		}
		for _, r := range s.CheckRuns.Nodes {
			run := CheckRun{
				Name:       string(r.Name),
				Status:     string(r.Status),
				Conclusion: string(r.Conclusion),
			}
			if r.DetailsUrl.URL != nil {
				run.DetailsUrl = r.DetailsUrl.URL.String()
			}
			if !r.StartedAt.IsZero() {
				run.StartedAt = r.StartedAt.Format(time.RFC3339)
			}
			if !r.CompletedAt.IsZero() {
				run.CompletedAt = r.CompletedAt.Format(time.RFC3339)
			}
			suite.Runs = append(suite.Runs, run)
		}
		out.Suites = append(out.Suites, suite)
	}
	return out, nil
}

// githubRepoConfig is the handful of bridge settings FetchCheckStatus needs:
// who we auth as, and which GitHub repo we're talking about.
type githubRepoConfig struct {
	owner   string
	project string
	login   string
}

func readGithubConfig(repo *cache.RepoCache) (*githubRepoConfig, error) {
	kv, err := repo.LocalConfig().ReadAll("git-bug.bridge.github.")
	if err != nil || len(kv) == 0 {
		return nil, nil
	}
	cfg := &githubRepoConfig{
		owner:   kv["git-bug.bridge.github."+confKeyOwner],
		project: kv["git-bug.bridge.github."+confKeyProject],
		login:   kv["git-bug.bridge.github."+confKeyDefaultLogin],
	}
	if cfg.owner == "" || cfg.project == "" {
		return nil, fmt.Errorf("github bridge missing %s/%s in config", confKeyOwner, confKeyProject)
	}
	return cfg, nil
}
