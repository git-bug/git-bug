package validate

import (
	"fmt"
	"testing"

	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/identity"
	"github.com/MichaelMure/git-bug/repository"
	"github.com/stretchr/testify/require"
)

func checkAddIdentity(t *testing.T, backend *cache.RepoCache, name, email, armoredPubkey string) *cache.IdentityCache {
	key, err := identity.NewKey(armoredPubkey)
	require.NoError(t, err)

	id, err := backend.NewIdentityWithKeyRaw(name, email, "", "", nil, key)
	require.NoError(t, err)

	return id
}

func checkValidator(t *testing.T, backend *cache.RepoCache, errorMsg, firstKey string) {
	validator, err := NewValidator(backend)
	if errorMsg == "" {
		require.NoError(t, err)
		if firstKey == "" {
			require.Nil(t, validator.FirstKey)
		} else {
			require.Equal(t, firstKey, validator.FirstKey.ArmoredPublicKey)
		}
	} else {
		require.EqualError(t, err, errorMsg)
	}
}

func checkAddKey(t *testing.T, id *cache.IdentityCache, armoredKey string) {
	key, err := identity.NewKey(armoredKey)
	require.NoError(t, err)

	err = id.Mutate(func(m identity.Mutator) identity.Mutator {
		m.Keys = append(m.Keys, key)
		return m
	})
	require.NoError(t, err)

	err = id.Commit()
	require.NoError(t, err)
}

func TestNewValidator_EmptyRepo(t *testing.T) {
	repo := repository.CreateTestRepo(false)
	defer repository.CleanupTestRepos(t, repo)

	backend, err := cache.NewRepoCache(repo)
	require.NoError(t, err)

	checkValidator(t, backend, "", "")
	validator, err := NewValidator(backend)
	require.NoError(t, err)
	require.Nil(t, validator.FirstKey)
}

func TestNewValidator_OneIdentity(t *testing.T) {
	repo := repository.CreateTestRepo(false)
	defer repository.CleanupTestRepos(t, repo)

	backend, err := cache.NewRepoCache(repo)
	require.NoError(t, err)

	armoredPubkey := repository.SetupSigningKey(t, repo, "a@e.org")

	_ = checkAddIdentity(t, backend, "A", "a@e.org", armoredPubkey)
	checkValidator(t, backend, "", armoredPubkey)
}

func TestNewValidator_TwoSeparateIdentities(t *testing.T) {
	repo := repository.CreateTestRepo(false)
	defer repository.CleanupTestRepos(t, repo)

	backend, err := cache.NewRepoCache(repo)
	require.NoError(t, err)

	armoredPubkey := repository.SetupSigningKey(t, repo, "a@e.org")
	_ = checkAddIdentity(t, backend, "A", "a@e.org", armoredPubkey)

	armoredPubkey2 := repository.SetupSigningKey(t, repo, "b@e.org")
	id2 := checkAddIdentity(t, backend, "B", "b@e.org", armoredPubkey2)

	msg := fmt.Sprintf("failed to validate identities: invalid identity %s (%s): invalid signature for commit %s: no key can verify the signature",
		id2.Id(), id2.Email(), id2.Versions()[0].CommitHash())
	checkValidator(t, backend, msg, "")
}

func TestNewValidator_IdentityWithSameKeyTwice(t *testing.T) {
	repo := repository.CreateTestRepo(false)
	defer repository.CleanupTestRepos(t, repo)

	backend, err := cache.NewRepoCache(repo)
	require.NoError(t, err)

	armoredPubkey := repository.SetupSigningKey(t, repo, "a@e.org")
	id1 := checkAddIdentity(t, backend, "A", "a@e.org", armoredPubkey)

	checkAddKey(t, id1, armoredPubkey)

	msg := fmt.Sprintf("failed to read identity versions: keys with identical keyId introduced in commits %s and %s",
		id1.Versions()[0].CommitHash(), id1.Versions()[1].CommitHash())
	checkValidator(t, backend, msg, "")
}

func TestNewValidator_TwoIdentitiesWithSameKey(t *testing.T) {
	repo := repository.CreateTestRepo(false)
	defer repository.CleanupTestRepos(t, repo)

	backend, err := cache.NewRepoCache(repo)
	require.NoError(t, err)

	armoredPubkey := repository.SetupSigningKey(t, repo, "a@e.org")
	id1 := checkAddIdentity(t, backend, "A", "a@e.org", armoredPubkey)

	id2 := checkAddIdentity(t, backend, "B", "b@e.org", armoredPubkey)

	_, err = NewValidator(backend)
	require.EqualError(t, err,
		fmt.Sprintf("failed to read identity versions: keys with identical keyId introduced in commits %s and %s",
			id1.Versions()[0].CommitHash(), id2.Versions()[0].CommitHash()))
}

func TestNewValidator_TwoIdentitiesTwoVersions(t *testing.T) {
	repo := repository.CreateTestRepo(false)
	defer repository.CleanupTestRepos(t, repo)

	backend, err := cache.NewRepoCache(repo)
	require.NoError(t, err)

	armoredPubkey := repository.SetupSigningKey(t, repo, "a@e.org")
	id1 := checkAddIdentity(t, backend, "A", "a@e.org", armoredPubkey)
	checkValidator(t, backend, "", armoredPubkey)

	armoredPubkey2 := repository.CreatePubkey(t)
	id2 := checkAddIdentity(t, backend, "B", "b@e.org", armoredPubkey2)

	armoredPubkey3 := repository.CreatePubkey(t)
	checkAddKey(t, id1, armoredPubkey3)

	armoredPubkey4 := repository.CreatePubkey(t)
	checkAddKey(t, id2, armoredPubkey4)

	checkValidator(t, backend, "", armoredPubkey)
}

func TestNewValidator_WrongEmail(t *testing.T) {
	repo := repository.CreateTestRepo(false)
	defer repository.CleanupTestRepos(t, repo)

	backend, err := cache.NewRepoCache(repo)
	require.NoError(t, err)

	armoredPubkey := repository.SetupSigningKey(t, repo, "a@e.org")
	repository.SetupKey(t, repo, "x@a.org", "", "")
	id1 := checkAddIdentity(t, backend, "A", "a@e.org", armoredPubkey)

	msg := fmt.Sprintf("failed to validate identities: invalid identity %s (%s): invalid signature for commit %s: git commit committer-email does not match the identity-email: x@a.org vs a@e.org",
		id1.Id(), id1.Email(), id1.Versions()[0].CommitHash())
	checkValidator(t, backend, msg, armoredPubkey)
}