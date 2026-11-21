package gitlab

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	gitlab "gitlab.com/gitlab-org/api/client-go"

	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/entity/dag"
	"github.com/git-bug/git-bug/repository/repotest"

	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/bridge/core"
	"github.com/git-bug/git-bug/bridge/core/auth"
	"github.com/git-bug/git-bug/cache"
)

const (
	testRepoBaseName = "git-bug-test-gitlab-exporter"
)

type testCase struct {
	name     string
	bug      *cache.BugCache
	numOp    int // number of original operations
	numOpExp int // number of operations after export
	numOpImp int // number of operations after import
}

func testCases(t *testing.T, repo *cache.RepoCache) []*testCase {
	// simple bug
	simpleBug, _, err := repo.Bugs().New(t.Context(), "simple bug", "new bug")
	require.NoError(t, err)
	require.NoError(t, simpleBug.Commit(t.Context()))

	// bug with comments
	bugWithComments, _, err := repo.Bugs().New(t.Context(), "bug with comments", "new bug")
	require.NoError(t, err)
	require.NoError(t, bugWithComments.Commit(t.Context()))

	_, _, err = bugWithComments.AddComment(t.Context(), "new comment")
	require.NoError(t, err)
	require.NoError(t, bugWithComments.Commit(t.Context()))

	// bug with label changes
	bugLabelChange, _, err := repo.Bugs().New(t.Context(), "bug label change", "new bug")
	require.NoError(t, err)
	require.NoError(t, bugLabelChange.Commit(t.Context()))

	_, _, err = bugLabelChange.ChangeLabels(t.Context(), []string{"bug"}, nil)
	require.NoError(t, err)

	_, _, err = bugLabelChange.ChangeLabels(t.Context(), []string{"core"}, nil)
	require.NoError(t, err)

	_, _, err = bugLabelChange.ChangeLabels(t.Context(), nil, []string{"bug"})
	require.NoError(t, err)
	require.NoError(t, bugLabelChange.Commit(t.Context()))

	// bug with comments editions
	bugWithCommentEditions, createOp, err := repo.Bugs().New(t.Context(), "bug with comments editions", "new bug")
	require.NoError(t, err)
	require.NoError(t, bugWithCommentEditions.Commit(t.Context()))

	_, err = bugWithCommentEditions.EditComment(
		t.Context(), entity.CombineIds(bugWithCommentEditions.Id(), createOp.Id()), "first comment edited")
	require.NoError(t, err)

	commentId, _, err := bugWithCommentEditions.AddComment(t.Context(), "first comment")
	require.NoError(t, err)

	_, err = bugWithCommentEditions.EditComment(t.Context(), commentId, "first comment edited")
	require.NoError(t, err)
	require.NoError(t, bugWithCommentEditions.Commit(t.Context()))

	// bug status changed
	bugStatusChanged, _, err := repo.Bugs().New(t.Context(), "bug status changed", "new bug")
	require.NoError(t, err)
	require.NoError(t, bugStatusChanged.Commit(t.Context()))

	_, err = bugStatusChanged.Close(t.Context())
	require.NoError(t, err)

	_, err = bugStatusChanged.Open(t.Context())
	require.NoError(t, err)
	require.NoError(t, bugStatusChanged.Commit(t.Context()))

	// bug title changed
	bugTitleEdited, _, err := repo.Bugs().New(t.Context(), "bug title edited", "new bug")
	require.NoError(t, err)
	require.NoError(t, bugTitleEdited.Commit(t.Context()))

	_, err = bugTitleEdited.SetTitle(t.Context(), "bug title edited again")
	require.NoError(t, err)
	require.NoError(t, bugTitleEdited.Commit(t.Context()))

	return []*testCase{
		{
			name:     "simple bug",
			bug:      simpleBug,
			numOp:    1,
			numOpExp: 2,
			numOpImp: 1,
		},
		{
			name:     "bug with comments",
			bug:      bugWithComments,
			numOp:    2,
			numOpExp: 4,
			numOpImp: 2,
		},
		{
			name:     "bug label change",
			bug:      bugLabelChange,
			numOp:    4,
			numOpExp: 8,
			numOpImp: 4,
		},
		{
			name:     "bug with comment editions",
			bug:      bugWithCommentEditions,
			numOp:    4,
			numOpExp: 8,
			numOpImp: 2,
		},
		{
			name:     "bug changed status",
			bug:      bugStatusChanged,
			numOp:    3,
			numOpExp: 6,
			numOpImp: 3,
		},
		{
			name:     "bug title edited",
			bug:      bugTitleEdited,
			numOp:    2,
			numOpExp: 4,
			numOpImp: 2,
		},
	}
}

func TestGitlabPushPull(t *testing.T) {
	// token must have 'repo' and 'delete_repo' scopes
	envToken := os.Getenv("GITLAB_API_TOKEN")
	if envToken == "" {
		t.Skip("Env var GITLAB_API_TOKEN missing")
	}

	// create repo backend
	repo := repotest.NewGoGitRepo(t, false)

	backend, err := cache.NewRepoCacheNoEvents(repo, nil)
	require.NoError(t, err)
	defer backend.Close()

	// set author identity
	login := "test-identity"
	author, err := backend.Identities().New(t.Context(), "test identity", "test@test.org")
	require.NoError(t, err)
	author.SetMetadata(metaKeyGitlabLogin, login)
	err = author.Commit(t.Context())
	require.NoError(t, err)

	err = backend.SetUserIdentity(t.Context(), author)
	require.NoError(t, err)

	token := auth.NewToken(target, envToken)
	token.SetMetadata(auth.MetaKeyLogin, login)
	token.SetMetadata(auth.MetaKeyBaseURL, defaultBaseURL)
	err = auth.Store(repo, token)
	require.NoError(t, err)

	tests := testCases(t, backend)

	// On interrupt, cancel ctx instead of killing the process, so that the test
	// fails normally and the cleanups below (removing the Gitlab repository) run.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	t.Cleanup(stop)

	// generate project name
	projectName := generateRepoName()

	// create target Gitlab repository
	projectID, err := createRepository(ctx, projectName, token)
	require.NoError(t, err)

	fmt.Printf("created project: %s (%d)\n", projectName, projectID)

	// Make sure to remove the Gitlab repository when the test end.
	// ctx is not used, as it's already canceled after an interrupt.
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
		defer cancel()
		if err := deleteRepository(ctx, projectID, token); err != nil {
			t.Error(err)
			return
		}
		fmt.Printf("deleted repository: %s (%d)\n", projectName, projectID)
	})

	// initialize exporter
	exporter := &gitlabExporter{}
	err = exporter.Init(ctx, backend, core.Configuration{
		confKeyProjectID:     strconv.FormatInt(projectID, 10),
		confKeyGitlabBaseUrl: defaultBaseURL,
		confKeyDefaultLogin:  login,
	})
	require.NoError(t, err)

	start := time.Now()

	// export all bugs
	exportEvents, err := exporter.ExportAll(ctx, backend, time.Time{})
	require.NoError(t, err)

	for result := range exportEvents {
		require.NoError(t, result.Err)
	}
	require.NoError(t, err)

	fmt.Printf("repository exported in %f seconds\n", time.Since(start).Seconds())

	repoTwo := repotest.NewGoGitRepo(t, false)

	// create a second backend
	backendTwo, err := cache.NewRepoCacheNoEvents(repoTwo, nil)
	require.NoError(t, err)
	defer backendTwo.Close()

	importer := &gitlabImporter{}
	err = importer.Init(ctx, backend, core.Configuration{
		confKeyProjectID:     strconv.FormatInt(projectID, 10),
		confKeyGitlabBaseUrl: defaultBaseURL,
		confKeyDefaultLogin:  login,
	})
	require.NoError(t, err)

	// import all exported bugs to the second backend
	importEvents, err := importer.ImportAll(ctx, backendTwo, time.Time{})
	require.NoError(t, err)

	for result := range importEvents {
		require.NoError(t, result.Err)
	}

	require.Len(t, backendTwo.Bugs().AllIds(), len(tests))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// for each operation a SetMetadataOperation will be added
			// so number of operations should double
			require.Len(t, tt.bug.Snapshot().Operations, tt.numOpExp)

			// verify operation have correct metadata
			for _, op := range tt.bug.Snapshot().Operations {
				// Check if the originals operations (*not* SetMetadata) are tagged properly
				if _, ok := op.(dag.OperationDoesntChangeSnapshot); !ok {
					_, haveIDMetadata := op.GetMetadata(metaKeyGitlabId)
					require.True(t, haveIDMetadata)

					_, haveURLMetada := op.GetMetadata(metaKeyGitlabUrl)
					require.True(t, haveURLMetada)
				}
			}

			// get bug gitlab ID
			bugGitlabID, ok := tt.bug.Snapshot().GetCreateMetadata(metaKeyGitlabId)
			require.True(t, ok)

			// retrieve bug from backendTwo
			importedBug, err := backendTwo.Bugs().ResolveBugCreateMetadata(t.Context(), metaKeyGitlabId, bugGitlabID)
			require.NoError(t, err)

			// verify bug have same number of original operations
			require.Len(t, importedBug.Snapshot().Operations, tt.numOpImp)

			// verify bugs are tagged with origin=gitlab
			issueOrigin, ok := importedBug.Snapshot().GetCreateMetadata(core.MetaKeyOrigin)
			require.True(t, ok)
			require.Equal(t, issueOrigin, target)

			// TODO: maybe more tests to ensure bug final state
		})
	}
}

func generateRepoName() string {
	var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]rune, 8)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return fmt.Sprintf("%s-%s", testRepoBaseName, string(b))
}

// create repository need a token with scope 'repo'
func createRepository(ctx context.Context, name string, token *auth.Token) (int64, error) {
	client, err := buildClient(defaultBaseURL, token)
	if err != nil {
		return 0, err
	}

	project, _, err := client.Projects.CreateProject(
		&gitlab.CreateProjectOptions{
			Name: gitlab.Ptr(name),
		},
		gitlab.WithContext(ctx),
	)
	if err != nil {
		return 0, err
	}

	// fmt.Println("Project URL:", project.WebURL)

	return project.ID, nil
}

// delete repository need a token with scope 'delete_repo'
func deleteRepository(ctx context.Context, project int64, token *auth.Token) error {
	client, err := buildClient(defaultBaseURL, token)
	if err != nil {
		return err
	}

	_, err = client.Projects.DeleteProject(project, nil, gitlab.WithContext(ctx))
	return err
}

func TestGitlabExportCrossBridgeBaseURLAndLabels(t *testing.T) {
	var createdIssues int
	var lastUpdatedLabels []string

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v4/projects/999/issues", func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		createdIssues++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":      1001,
			"iid":     42,
			"web_url": "http://example.com/group/proj/-/issues/42",
		})
	})
	mux.HandleFunc("/api/v4/projects/999/issues/42", func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPut, r.Method)
		var body struct {
			Labels string `json:"labels"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		lastUpdatedLabels = strings.Split(body.Labels, ",")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":      1001,
			"iid":     42,
			"web_url": "http://example.com/group/proj/-/issues/42",
		})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	repo := repotest.NewGoGitRepo(t, false)
	backend, err := cache.NewRepoCacheNoEvents(repo, nil)
	require.NoError(t, err)
	defer func() { _ = backend.Close() }()

	ctx := context.Background()
	author, err := backend.Identities().New(ctx, "test identity", "test@test.org")
	require.NoError(t, err)
	author.SetMetadata(metaKeyGitlabLogin, "test-login")
	require.NoError(t, author.Commit(ctx))
	require.NoError(t, backend.SetUserIdentity(ctx, author))

	// Create bug with origin=github and one already-synced label ("bug"), plus one unsynced label ("enhancement")
	b, _, err := backend.Bugs().NewRaw(
		author,
		100,
		"cross-bridge issue",
		"issue body",
		nil,
		map[string]string{
			core.MetaKeyOrigin: "github",
		},
	)
	require.NoError(t, err)

	_, firstLabelOp, err := b.ChangeLabelsRaw(author, 101, []string{"bug"}, nil, nil)
	require.NoError(t, err)
	// Mark firstLabelOp as already synced to GitLab
	require.NoError(t, markOperationAsExported(ctx, b, firstLabelOp.Id(), "42", ""))

	_, _, err = b.ChangeLabelsRaw(author, 102, []string{"enhancement"}, nil, nil)
	require.NoError(t, err)
	require.NoError(t, b.Commit(ctx))

	token := auth.NewToken(target, "test-token")
	client, err := buildClient(server.URL, token)
	require.NoError(t, err)

	exporter := &gitlabExporter{
		conf: core.Configuration{
			confKeyProjectID:     "999",
			confKeyGitlabBaseUrl: server.URL,
			confKeyDefaultLogin:  "test-login",
		},
		identityClient: map[entity.Id]*gitlab.Client{
			author.Id(): client,
		},
		repositoryID:       "999",
		cachedOperationIDs: make(map[string]string),
	}

	out := make(chan core.ExportResult, 16)
	exporter.exportBug(ctx, b, out)
	close(out)
	for res := range out {
		require.NoError(t, res.Err)
	}

	require.Equal(t, 1, createdIssues)
	// Verify base URL metadata was properly stored on createOp
	baseURL, ok := b.Snapshot().GetCreateMetadata(metaKeyGitlabBaseUrl)
	require.True(t, ok)
	require.Equal(t, server.URL, baseURL)

	// Verify both the already-synced label ("bug") and new label ("enhancement") were sent to GitLab
	require.ElementsMatch(t, []string{"bug", "enhancement"}, lastUpdatedLabels)
}
