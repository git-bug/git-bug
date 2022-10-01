package cache

import (
	"bytes"
	"testing"

	"github.com/MichaelMure/git-bug/repository"
	"github.com/stretchr/testify/require"
)

const ExpectedCacheInitializationMessage = "Building identity cache... Done.\nBuilding bug cache... Done.\n"

func NewTestRepoCache(t *testing.T, repo repository.TestedRepo) (*RepoCache, *bytes.Buffer) {
	t.Helper()

	stderr := &bytes.Buffer{}
	cache, err := NewRepoCache(repo, stderr)
	require.NoError(t, err)
	require.Equal(t, ExpectedCacheInitializationMessage, stderr.String())

	stderr.Reset()

	return cache, stderr
}
