package github

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/MichaelMure/git-bug/bridge/core"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/repository"
	"github.com/MichaelMure/git-bug/util/interrupt"
	"github.com/stretchr/testify/require"
)

func TestExporter(t *testing.T) {
	//TODO test strategy
	tests := []struct {
		name string
	}{
		{
			name: "bug creation",
		},
	}

	repo := repository.CreateTestRepo(false)
	defer repository.CleanupTestRepos(t, repo)

	backend, err := cache.NewRepoCache(repo)
	require.NoError(t, err)

	defer backend.Close()
	interrupt.RegisterCleaner(backend.Close)

	token := os.Getenv("GITHUB_TOKEN_PRIVATE")
	if token == "" {
		t.Skip("Env var GITHUB_TOKEN_PRIVATE missing")
	}

	exporter := &githubExporter{}
	err = exporter.Init(core.Configuration{
		keyOwner:   "MichaelMure",
		keyProject: "git-bug-exporter-tests",
		keyToken:   token,
	})
	require.NoError(t, err)

	start := time.Now()

	err = exporter.ExportAll(backend, time.Time{})
	require.NoError(t, err)

	fmt.Printf("test repository exported in %f seconds\n", time.Since(start).Seconds())

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

		})
	}
}

func genRepoName() {}

func createRepository() {}

func deleteRepository() {}

// verifyIssue is full
// comments
func verifyIssue() {

}
