package bridge

import (
	"testing"
	"github.com/stretchr/testify/require"

	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/repository"
)

func TestDefaultBridge(t *testing.T) {
	repo := repository.CreateGoGitTestRepo(false)
	defer repository.CleanupTestRepos(repo)
	crepo, err := cache.NewRepoCache(repo)
	require.NoError(t, err)

	_, err = DefaultBridge(crepo)
	require.Error(t, err) // no bridges

	config := crepo.LocalConfig()
	require.NoError(t, config.StoreString("git-bug.bridge.github.target"       , "github"     ))
	require.NoError(t, config.StoreString("git-bug.bridge.github.project"      , "testproject"))
	require.NoError(t, config.StoreString("git-bug.bridge.github.owner"        , "testuser"   ))
	require.NoError(t, config.StoreString("git-bug.bridge.github.default-login", "somelogin"  ))
	_, err = DefaultBridge(crepo)
	require.NoError(t, err)
}
