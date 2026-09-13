package oauth

import (
	"testing"

	"github.com/markbates/goth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/cache"
	"github.com/git-bug/git-bug/repository"
)

func newTestCache(t *testing.T) *cache.RepoCacheIdentity {
	t.Helper()
	repo := repository.CreateGoGitTestRepo(t, false)
	rc, err := cache.NewRepoCacheNoEvents(repo)
	require.NoError(t, err)
	t.Cleanup(func() { _ = rc.Close() })
	return rc.Identities()
}

func makeGothUser(userID, name, email, login, avatar string) goth.User {
	return goth.User{
		UserID:    userID,
		Name:      name,
		Email:     email,
		NickName:  login,
		AvatarURL: avatar,
		Provider:  "github",
	}
}

func TestFindOrImport_CreatesOnFirstLogin(t *testing.T) {
	identities := newTestCache(t)
	user := makeGothUser("12345678", "Alice Example", "alice@example.com", "alice", "https://example.com/avatar.png")

	id, err := FindOrImport(identities, user, "github")
	require.NoError(t, err)
	require.NotNil(t, id)

	assert.Equal(t, "Alice Example", id.Name())
	assert.Equal(t, "alice@example.com", id.Email())
	assert.Equal(t, "alice", id.Login())
	// Immutable metadata must use the numeric user ID, not the login.
	assert.Equal(t, "12345678", id.ImmutableMetadata()["github:user-id"])
}

func TestFindOrImport_ReturnsExistingOnSecondLogin(t *testing.T) {
	identities := newTestCache(t)
	user := makeGothUser("12345678", "Alice Example", "alice@example.com", "alice", "")

	first, err := FindOrImport(identities, user, "github")
	require.NoError(t, err)

	second, err := FindOrImport(identities, user, "github")
	require.NoError(t, err)

	assert.Equal(t, first.Id(), second.Id(), "second login must return the same identity")
}

func TestFindOrImport_DifferentUsersGetDifferentIdentities(t *testing.T) {
	identities := newTestCache(t)

	alice := makeGothUser("11111111", "Alice", "alice@example.com", "alice", "")
	bob := makeGothUser("22222222", "Bob", "bob@example.com", "bob", "")

	idAlice, err := FindOrImport(identities, alice, "github")
	require.NoError(t, err)

	idBob, err := FindOrImport(identities, bob, "github")
	require.NoError(t, err)

	assert.NotEqual(t, idAlice.Id(), idBob.Id())
}

func TestFindOrImport_SameNumericIDDifferentLogin(t *testing.T) {
	// The login (username) may change; the numeric ID must be the stable lookup key.
	identities := newTestCache(t)

	first := makeGothUser("12345678", "Alice", "alice@example.com", "old-login", "")
	_, err := FindOrImport(identities, first, "github")
	require.NoError(t, err)

	renamed := makeGothUser("12345678", "Alice", "alice@example.com", "new-login", "")
	id, err := FindOrImport(identities, renamed, "github")
	require.NoError(t, err)

	assert.Equal(t, "12345678", id.ImmutableMetadata()["github:user-id"],
		"lookup must succeed even after username change")
}
