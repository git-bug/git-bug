package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/MichaelMure/git-bug/bridge/core"
	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/repository"
	"github.com/MichaelMure/git-bug/util/interrupt"
)

const (
	testRepoBaseName = "git-bug-test-github-exporter"
)

type testCase struct {
	name    string
	bug     *cache.BugCache
	numOrOp int // number of original operations
}

func testCases(repo *cache.RepoCache, identity *cache.IdentityCache) ([]*testCase, error) {
	// simple bug
	simpleBug, err := repo.NewBug("simple bug", "new bug")
	if err != nil {
		return nil, err
	}

	// bug with comments
	bugWithComments, err := repo.NewBug("bug with comments", "new bug")
	if err != nil {
		return nil, err
	}

	_, err = bugWithComments.AddComment("new comment")
	if err != nil {
		return nil, err
	}

	// bug with label changes
	bugLabelChange, err := repo.NewBug("bug label change", "new bug")
	if err != nil {
		return nil, err
	}

	_, _, err = bugLabelChange.ChangeLabels([]string{"bug"}, nil)
	if err != nil {
		return nil, err
	}

	_, _, err = bugLabelChange.ChangeLabels([]string{"core"}, nil)
	if err != nil {
		return nil, err
	}

	_, _, err = bugLabelChange.ChangeLabels(nil, []string{"bug"})
	if err != nil {
		return nil, err
	}

	// bug with comments editions
	bugWithCommentEditions, err := repo.NewBug("bug with comments editions", "new bug")
	if err != nil {
		return nil, err
	}

	createOpHash, err := bugWithCommentEditions.Snapshot().Operations[0].Hash()
	if err != nil {
		return nil, err
	}

	_, err = bugWithCommentEditions.EditComment(createOpHash, "first comment edited")
	if err != nil {
		return nil, err
	}

	commentOp, err := bugWithCommentEditions.AddComment("first comment")
	if err != nil {
		return nil, err
	}

	commentOpHash, err := commentOp.Hash()
	if err != nil {
		return nil, err
	}

	_, err = bugWithCommentEditions.EditComment(commentOpHash, "first comment edited")
	if err != nil {
		return nil, err
	}

	// bug status changed
	bugStatusChanged, err := repo.NewBug("bug status changed", "new bug")
	if err != nil {
		return nil, err
	}

	_, err = bugStatusChanged.Close()
	if err != nil {
		return nil, err
	}

	_, err = bugStatusChanged.Open()
	if err != nil {
		return nil, err
	}

	// bug title changed
	bugTitleEdited, err := repo.NewBug("bug title edited", "new bug")
	if err != nil {
		return nil, err
	}

	_, err = bugTitleEdited.SetTitle("bug title edited again")
	if err != nil {
		return nil, err
	}

	return []*testCase{
		&testCase{
			name:    "simple bug",
			bug:     simpleBug,
			numOrOp: 1,
		},
		&testCase{
			name:    "bug with comments",
			bug:     bugWithComments,
			numOrOp: 2,
		},
		&testCase{
			name:    "bug label change",
			bug:     bugLabelChange,
			numOrOp: 4,
		},
		&testCase{
			name:    "bug with comment editions",
			bug:     bugWithCommentEditions,
			numOrOp: 4,
		},
		&testCase{
			name:    "bug changed status",
			bug:     bugStatusChanged,
			numOrOp: 3,
		},
		&testCase{
			name:    "bug title edited",
			bug:     bugTitleEdited,
			numOrOp: 2,
		},
	}, nil

}

func TestPushPull(t *testing.T) {
	// repo owner
	user := os.Getenv("TEST_USER")

	// token must have 'repo' and 'delete_repo' scopes
	token := os.Getenv("GITHUB_TOKEN_ADMIN")
	if token == "" {
		t.Skip("Env var GITHUB_TOKEN_ADMIN missing")
	}

	// create repo backend
	repo := repository.CreateTestRepo(false)
	defer repository.CleanupTestRepos(t, repo)

	backend, err := cache.NewRepoCache(repo)
	require.NoError(t, err)

	// set author identity
	author, err := backend.NewIdentity("test identity", "test@test.org")
	if err != nil {
		t.Fatal(err)
	}

	err = backend.SetUserIdentity(author)
	if err != nil {
		t.Fatal(err)
	}

	defer backend.Close()
	interrupt.RegisterCleaner(backend.Close)

	tests, err := testCases(backend, author)
	if err != nil {
		t.Fatal(err)
	}

	// generate project name
	projectName := generateRepoName()

	// create target Github repository
	if err := createRepository(projectName, token); err != nil {
		t.Fatal(err)
	}
	fmt.Println("created repository", projectName)

	// Make sure to remove the Github repository when the test end
	defer func(t *testing.T) {
		if err := deleteRepository(projectName, user, token); err != nil {
			t.Fatal(err)
		}
		fmt.Println("deleted repository:", projectName)
	}(t)

	// initialize exporter
	exporter := &githubExporter{}
	err = exporter.Init(core.Configuration{
		keyOwner:   user,
		keyProject: projectName,
		keyToken:   token,
	})
	require.NoError(t, err)

	start := time.Now()

	// export all bugs
	err = exporter.ExportAll(backend, time.Time{})
	require.NoError(t, err)

	fmt.Printf("test repository exported in %f seconds\n", time.Since(start).Seconds())

	repoTwo := repository.CreateTestRepo(false)
	defer repository.CleanupTestRepos(t, repoTwo)

	// create a second backend
	backendTwo, err := cache.NewRepoCache(repoTwo)
	require.NoError(t, err)

	importer := &githubImporter{}
	err = importer.Init(core.Configuration{
		keyOwner:   user,
		keyProject: projectName,
		keyToken:   token,
	})
	require.NoError(t, err)

	// import all exported bugs to the second backend
	err = importer.ImportAll(backendTwo, time.Time{})
	require.NoError(t, err)

	require.Len(t, backendTwo.AllBugsIds(), len(tests))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// for each operation a SetMetadataOperation will be added
			// so number of operations should double
			require.Len(t, tt.bug.Snapshot().Operations, tt.numOrOp*2)

			// verify operation have correcte metadata
			for _, op := range tt.bug.Snapshot().Operations {
				// Check if the originals operations (*not* SetMetadata) are tagged properly
				if _, ok := op.(*bug.SetMetadataOperation); !ok {
					_, haveIDMetadata := op.GetMetadata(keyGithubId)
					require.True(t, haveIDMetadata)

					_, haveURLMetada := op.GetMetadata(keyGithubUrl)
					require.True(t, haveURLMetada)
				}
			}

			// get bug github ID
			bugGithubID, ok := tt.bug.Snapshot().Operations[0].GetMetadata(keyGithubId)
			require.True(t, ok)

			// retrive bug from backendTwo
			importedBug, err := backendTwo.ResolveBugCreateMetadata(keyGithubId, bugGithubID)
			require.NoError(t, err)

			// verify bug have same number of original operations
			require.Len(t, importedBug.Snapshot().Operations, tt.numOrOp)

			// verify bugs are taged with origin=github
			issueOrigin, ok := importedBug.Snapshot().Operations[0].GetMetadata(keyOrigin)
			require.True(t, ok)
			require.Equal(t, issueOrigin, target)

			//TODO: maybe more tests to ensure bug final state
		})
	}
}

func generateRepoName() string {
	rand.Seed(time.Now().UnixNano())
	var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]rune, 8)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return fmt.Sprintf("%s-%s", testRepoBaseName, string(b))
}

// create repository need a token with scope 'repo'
func createRepository(project, token string) error {
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

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
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

	return resp.Body.Close()
}

// delete repository need a token with scope 'delete_repo'
func deleteRepository(project, owner, token string) error {
// This function use the V3 Github API because repository removal is not supported yet on the V4 API.
	url := fmt.Sprintf("%s/repos/%s/%s", githubV3Url, owner, project)

	req, err := http.NewRequest("DELETE", url, nil)
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
