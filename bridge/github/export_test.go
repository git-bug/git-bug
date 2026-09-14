package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/bridge/core"
	"github.com/git-bug/git-bug/bridge/core/auth"
	"github.com/git-bug/git-bug/cache"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/entity/dag"
	"github.com/git-bug/git-bug/repository"
)

const (
	testRepoBaseName = "git-bug-test-github-exporter"
)

type testCase struct {
	name    string
	bug     *cache.BugCache
	numOrOp int // number of original operations
}

func testCases(t *testing.T, repo *cache.RepoCache) []*testCase {
	// simple bug
	simpleBug, _, err := repo.Bugs().New("simple bug", "new bug")
	require.NoError(t, err)

	// bug with comments
	bugWithComments, _, err := repo.Bugs().New("bug with comments", "new bug")
	require.NoError(t, err)

	_, _, err = bugWithComments.AddComment("new comment")
	require.NoError(t, err)

	// bug with label changes
	bugLabelChange, _, err := repo.Bugs().New("bug label change", "new bug")
	require.NoError(t, err)

	_, _, err = bugLabelChange.ChangeLabels([]string{"bug"}, nil)
	require.NoError(t, err)

	_, _, err = bugLabelChange.ChangeLabels([]string{"core"}, nil)
	require.NoError(t, err)

	_, _, err = bugLabelChange.ChangeLabels(nil, []string{"bug"})
	require.NoError(t, err)

	_, _, err = bugLabelChange.ChangeLabels([]string{"InVaLiD"}, nil)
	require.NoError(t, err)

	_, _, err = bugLabelChange.ChangeLabels([]string{"bUG"}, nil)
	require.NoError(t, err)

	// bug with comments editions
	bugWithCommentEditions, createOp, err := repo.Bugs().New("bug with comments editions", "new bug")
	require.NoError(t, err)

	_, err = bugWithCommentEditions.EditComment(
		entity.CombineIds(bugWithCommentEditions.Id(), createOp.Id()), "first comment edited")
	require.NoError(t, err)

	commentId, _, err := bugWithCommentEditions.AddComment("first comment")
	require.NoError(t, err)

	_, err = bugWithCommentEditions.EditComment(commentId, "first comment edited")
	require.NoError(t, err)

	// bug status changed
	bugStatusChanged, _, err := repo.Bugs().New("bug status changed", "new bug")
	require.NoError(t, err)

	_, err = bugStatusChanged.Close()
	require.NoError(t, err)

	_, err = bugStatusChanged.Open()
	require.NoError(t, err)

	// bug title changed
	bugTitleEdited, _, err := repo.Bugs().New("bug title edited", "new bug")
	require.NoError(t, err)

	_, err = bugTitleEdited.SetTitle("bug title edited again")
	require.NoError(t, err)

	return []*testCase{
		{
			name:    "simple bug",
			bug:     simpleBug,
			numOrOp: 1,
		},
		{
			name:    "bug with comments",
			bug:     bugWithComments,
			numOrOp: 2,
		},
		{
			name:    "bug label change",
			bug:     bugLabelChange,
			numOrOp: 6,
		},
		{
			name:    "bug with comment editions",
			bug:     bugWithCommentEditions,
			numOrOp: 4,
		},
		{
			name:    "bug changed status",
			bug:     bugStatusChanged,
			numOrOp: 3,
		},
		{
			name:    "bug title edited",
			bug:     bugTitleEdited,
			numOrOp: 2,
		},
	}
}

func TestGithubPushPull(t *testing.T) {
	// repo owner
	envUser := os.Getenv("GITHUB_USER")
	if envUser == "" {
		t.Skip("missing required environment variable: GITHUB_USER")
	}

	// token must have 'repo' and 'delete_repo' scopes
	envToken := os.Getenv("GITHUB_TOKEN")
	if envToken == "" {
		t.Skip("missing required environment variable: GITHUB_TOKEN")
	}

	// create repo backend
	repo := repository.CreateGoGitTestRepo(t, false)

	backend, err := cache.NewRepoCacheNoEvents(repo)
	require.NoError(t, err)

	// set author identity
	login := "identity-test"
	author, err := backend.Identities().New("test identity", "test@test.org")
	require.NoError(t, err)
	author.SetMetadata(metaKeyGithubLogin, login)
	err = author.Commit()
	require.NoError(t, err)

	err = backend.SetUserIdentity(author)
	require.NoError(t, err)

	defer backend.Close()

	// Setup token + cleanup
	token := auth.NewToken(target, envToken)
	token.SetMetadata(auth.MetaKeyLogin, login)
	err = auth.Store(repo, token)
	require.NoError(t, err)

	defer auth.Remove(repo, token.ID())

	tests := testCases(t, backend)

	// On interrupt, cancel ctx instead of killing the process, so that the test
	// fails normally and the cleanups below (removing the Github repository) run.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	t.Cleanup(stop)

	// generate project name
	projectName := generateRepoName()

	// create target Github repository
	err = createRepository(ctx, projectName, envToken)
	require.NoError(t, err)

	slog.Info("created github repository", "name", projectName)

	// Make sure to remove the Github repository when the test end.
	// ctx is not used, as it's already canceled after an interrupt.
	t.Cleanup(func() {
		if err := deleteRepository(context.Background(), projectName, envUser, envToken); err != nil {
			t.Error(err)
			return
		}
		fmt.Println("deleted repository:", projectName)
	})

	// Let Github handle the repo creation and update all their internal caches.
	// Avoid HTTP error 404 retrieving repository node id
	err = waitRepository(ctx, projectName, envUser, envToken, 60*time.Second)
	require.NoError(t, err)

	// initialize exporter
	exporter := &githubExporter{}
	err = exporter.Init(ctx, backend, core.Configuration{
		confKeyOwner:        envUser,
		confKeyProject:      projectName,
		confKeyDefaultLogin: login,
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

	fmt.Printf("test repository exported in %f seconds\n", time.Since(start).Seconds())

	repoTwo := repository.CreateGoGitTestRepo(t, false)

	// create a second backend
	backendTwo, err := cache.NewRepoCacheNoEvents(repoTwo)
	require.NoError(t, err)

	importer := &githubImporter{}
	err = importer.Init(ctx, backend, core.Configuration{
		confKeyOwner:        envUser,
		confKeyProject:      projectName,
		confKeyDefaultLogin: login,
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
			require.Len(t, tt.bug.Snapshot().Operations, tt.numOrOp*2)

			// verify operation have correct metadata
			for _, op := range tt.bug.Snapshot().Operations {
				// Check if the originals operations (*not* SetMetadata) are tagged properly
				if _, ok := op.(dag.OperationDoesntChangeSnapshot); !ok {
					_, haveIDMetadata := op.GetMetadata(metaKeyGithubId)
					require.True(t, haveIDMetadata)

					_, haveURLMetada := op.GetMetadata(metaKeyGithubUrl)
					require.True(t, haveURLMetada)
				}
			}

			// get bug github ID
			bugGithubID, ok := tt.bug.Snapshot().GetCreateMetadata(metaKeyGithubId)
			require.True(t, ok)

			// retrieve bug from backendTwo
			importedBug, err := backendTwo.Bugs().ResolveBugCreateMetadata(metaKeyGithubId, bugGithubID)
			require.NoError(t, err)

			// verify bug have same number of original operations
			require.Len(t, importedBug.Snapshot().Operations, tt.numOrOp)

			// verify bugs are tagged with origin=github
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
func createRepository(ctx context.Context, project, token string) error {
	// This function use the V3 Github API because repository creation is not supported yet on the V4 API.
	url := fmt.Sprintf("%s/user/repos", githubV3Url)

	params := struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Private     bool   `json:"private"`
		HasIssues   bool   `json:"has_issues"`
	}{
		Name:        project,
		Description: "git-bug exporter temporary test repository",
		Private:     true,
		HasIssues:   true,
	}

	data, err := json.Marshal(params)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(data))
	if err != nil {
		return err
	}

	// need the token for private repositories
	req.Header.Set("Authorization", fmt.Sprintf("token %s", token))

	client := &http.Client{
		Timeout: defaultTimeout,
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("error creating repository: %s: %s", resp.Status, body)
	}

	return nil
}

// waitRepository polls the V3 API until the repository is visible, as Github
// can take a moment before a freshly created repository is usable.
func waitRepository(ctx context.Context, project, owner, token string, timeout time.Duration) error {
	url := fmt.Sprintf("%s/repos/%s/%s", githubV3Url, owner, project)

	client := &http.Client{
		Timeout: defaultTimeout,
	}

	deadline := time.Now().Add(timeout)
	for {
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", fmt.Sprintf("token %s", token))

		resp, err := client.Do(req)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
			err = fmt.Errorf("unexpected status: %s", resp.Status)
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("repository %s/%s not available after %s: %w", owner, project, timeout, err)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
}

// delete repository need a token with scope 'delete_repo'
func deleteRepository(ctx context.Context, project, owner, token string) error {
	// This function use the V3 Github API because repository removal is not supported yet on the V4 API.
	url := fmt.Sprintf("%s/repos/%s/%s", githubV3Url, owner, project)

	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return err
	}

	// need the token for private repositories
	req.Header.Set("Authorization", fmt.Sprintf("token %s", token))

	client := &http.Client{
		Timeout: defaultTimeout,
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("error deleting repository")
	}

	return nil
}
