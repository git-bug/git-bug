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
	key, err := identity.NewKeyFromArmored(armoredPubkey)
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
			require.Equal(t, firstKey, validator.FirstKey.Armored())
		}
	} else {
		require.EqualError(t, err, errorMsg)
	}
}

func checkAddKey(t *testing.T, id *cache.IdentityCache, armoredKey string) {
	key, err := identity.NewKeyFromArmored(armoredKey)
	require.NoError(t, err)

	err = id.Mutate(identity.AddKeyMutator(key))
	require.NoError(t, err)

	err = id.Commit()
	require.NoError(t, err)
}

func checkRemoveKey(t *testing.T, id *cache.IdentityCache, armoredKey string) {
	key, err := identity.NewKeyFromArmored(armoredKey)
	require.NoError(t, err)

	var removedKey *identity.Key
	err = id.Mutate(identity.RemoveKeyMutator(key.Fingerprint(), &removedKey))
	require.NoError(t, err)

	require.Equal(t, key.Fingerprint(), removedKey.Fingerprint())

	err = id.Commit()
	require.NoError(t, err)
}

func TestNewValidator_EmptyRepo(t *testing.T) {
	repo := repository.CreateTestRepo(false)
	defer repository.CleanupTestRepos(repo)

	backend, err := cache.NewRepoCache(repo)
	require.NoError(t, err)

	checkValidator(t, backend, "", "")
	validator, err := NewValidator(backend)
	require.NoError(t, err)
	require.Nil(t, validator.FirstKey)
}

func TestNewValidator_OneIdentity(t *testing.T) {
	repo := repository.CreateTestRepo(false)
	defer repository.CleanupTestRepos(repo)

	backend, err := cache.NewRepoCache(repo)
	require.NoError(t, err)

	armoredPubkey := repository.SetupSigningKey(t, repo, "a@e.org")

	_ = checkAddIdentity(t, backend, "A", "a@e.org", armoredPubkey)
	checkValidator(t, backend, "", armoredPubkey)
}

func TestNewValidator_TwoSeparateIdentities(t *testing.T) {
	repo := repository.CreateTestRepo(false)
	defer repository.CleanupTestRepos(repo)

	backend, err := cache.NewRepoCache(repo)
	require.NoError(t, err)

	armoredPubkey := repository.SetupSigningKey(t, repo, "a@e.org")
	_ = checkAddIdentity(t, backend, "A", "a@e.org", armoredPubkey)

	armoredPubkey2 := repository.SetupSigningKey(t, repo, "b@e.org")
	id2 := checkAddIdentity(t, backend, "B", "b@e.org", armoredPubkey2)

	msg := fmt.Sprintf("failed to validate identities: invalid identity %s (%s) commit %s: invalid signature: no key can verify the signature",
		id2.Id(), id2.Email(), id2.Versions()[0].CommitHash())
	checkValidator(t, backend, msg, "")
}

func TestNewValidator_IdentityWithSameKeyTwice(t *testing.T) {
	repo := repository.CreateTestRepo(false)
	defer repository.CleanupTestRepos(repo)

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
	defer repository.CleanupTestRepos(repo)

	backend, err := cache.NewRepoCache(repo)
	require.NoError(t, err)

	armoredPubkey := repository.SetupSigningKey(t, repo, "a@e.org")
	id1 := checkAddIdentity(t, backend, "A", "a@e.org", armoredPubkey)

	id2 := checkAddIdentity(t, backend, "B", "b@e.org", armoredPubkey)

	_, err = NewValidator(backend)
	err1 := fmt.Sprintf("failed to read identity versions: keys with identical keyId introduced in commits %s and %s",
		id1.Versions()[0].CommitHash(), id2.Versions()[0].CommitHash())
	err2 := fmt.Sprintf("failed to read identity versions: keys with identical keyId introduced in commits %s and %s",
		id2.Versions()[0].CommitHash(), id1.Versions()[0].CommitHash())
	if err.Error() != err1 && err.Error() != err2 {
		t.Fatalf("Expected a different error, not %s", err.Error())
	}
}

func TestNewValidator_TwoIdentitiesTwoVersions(t *testing.T) {
	repo := repository.CreateTestRepo(false)
	defer repository.CleanupTestRepos(repo)

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
	defer repository.CleanupTestRepos(repo)

	backend, err := cache.NewRepoCache(repo)
	require.NoError(t, err)

	armoredPubkey := repository.SetupSigningKey(t, repo, "a@e.org")
	repository.SetupKey(t, repo, "x@a.org", "", "")
	id1 := checkAddIdentity(t, backend, "A", "a@e.org", armoredPubkey)

	msg := fmt.Sprintf("failed to validate identities: invalid identity %s (%s) commit %s: invalid signature: git commit committer-email does not match the identity-email: x@a.org vs a@e.org",
		id1.Id(), id1.Email(), id1.Versions()[0].CommitHash())
	checkValidator(t, backend, msg, armoredPubkey)
}

func TestNewValidator_RemovedKey(t *testing.T) {
	repo := repository.CreateTestRepo(false)
	defer repository.CleanupTestRepos(repo)

	backend, err := cache.NewRepoCache(repo)
	require.NoError(t, err)

	armoredPubkey := repository.SetupSigningKey(t, repo, "a@e.org")
	id1 := checkAddIdentity(t, backend, "A", "a@e.org", armoredPubkey)

	keyId2, armoredPubkey2, gpgWrapper2 := repository.CreateKey(t, "a@e.org")
	checkAddKey(t, id1, armoredPubkey2)
	checkValidator(t, backend, "", armoredPubkey)

	repository.SetupKey(t, repo, "a@e.org", keyId2, gpgWrapper2)
	checkRemoveKey(t, id1, armoredPubkey)
	checkValidator(t, backend, "", armoredPubkey)

	checkRemoveKey(t, id1, armoredPubkey2)
	checkValidator(t, backend, "", armoredPubkey)
}
