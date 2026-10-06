package core

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/cache"
	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/repository"
)

func TestFinishConfigDanglingUserIdentity(t *testing.T) {
	repo := repository.CreateGoGitTestRepo(t, false)

	// Point the user identity to an identity that doesn't exist in the repo,
	// as happens when the identity has been removed or the config copied over.
	dangling := entity.Id(strings.Repeat("a", 64))
	require.NoError(t, repo.LocalConfig().StoreString("git-bug.identity", dangling.String()))

	backend, err := cache.NewRepoCacheNoEvents(repo, nil)
	require.NoError(t, err)
	defer backend.Close()

	const metaKey = "test-login"

	err = FinishConfig(backend, metaKey, "jdoe")
	require.NoError(t, err)

	// a new identity tagged with the login must be the current user now
	user, err := backend.GetUserIdentity()
	require.NoError(t, err)
	require.NotEqual(t, dangling, user.Id())
	require.Equal(t, "jdoe", user.ImmutableMetadata()[metaKey])

	id, err := identity.GetUserIdentityId(repo)
	require.NoError(t, err)
	require.Equal(t, user.Id(), id)
}

func TestFinishConfigDanglingUserIdentityWithExisting(t *testing.T) {
	repo := repository.CreateGoGitTestRepo(t, false)

	backend, err := cache.NewRepoCacheNoEvents(repo, nil)
	require.NoError(t, err)
	defer backend.Close()

	const metaKey = "test-login"

	// Create an identity tagged with the login metadata in the repository
	existing, err := backend.Identities().NewFromGitUserRaw(map[string]string{
		metaKey: "jdoe",
	})
	require.NoError(t, err)

	// Set a dangling user identity in git config
	dangling := entity.Id(strings.Repeat("a", 64))
	require.NoError(t, repo.LocalConfig().StoreString("git-bug.identity", dangling.String()))

	err = FinishConfig(backend, metaKey, "jdoe")
	require.NoError(t, err)

	// the existing identity must have replaced the dangling one as the current user
	user, err := backend.GetUserIdentity()
	require.NoError(t, err)
	require.Equal(t, existing.Id(), user.Id())

	id, err := identity.GetUserIdentityId(repo)
	require.NoError(t, err)
	require.Equal(t, existing.Id(), id)
}
