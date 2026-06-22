package identity

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/repository"
	"github.com/git-bug/git-bug/util/lamport"
	"github.com/git-bug/git-bug/util/timestamp"
)

type mockIdentity struct {
	name  string
	login string
	email string
}

func (m *mockIdentity) Name() string {
	return m.name
}

func (m *mockIdentity) Login() string {
	return m.login
}

func (m *mockIdentity) Email() string {
	return m.email
}

func (m *mockIdentity) DisplayName() string {
	return m.name
}

func (m *mockIdentity) AvatarUrl() string {
	return ""
}

func (m *mockIdentity) Keys() []*Key {
	return nil
}

func (m *mockIdentity) SigningKey(repo repository.RepoKeyring) (*Key, error) {
	return nil, nil
}

func (m *mockIdentity) ValidKeysAtTime(clockName string, time lamport.Time) []*Key {
	return nil
}

func (m *mockIdentity) LastModification() timestamp.Timestamp {
	return 0
}

func (m *mockIdentity) LastModificationLamports() map[string]lamport.Time {
	return nil
}

func (m *mockIdentity) IsProtected() bool {
	return false
}

func (m *mockIdentity) Validate() error {
	return nil
}

func (m *mockIdentity) NeedCommit() bool {
	return false
}

func (m *mockIdentity) Id() entity.Id {
	return ""
}

// identitiesEqual compares two identities by their versions
func identitiesEqual(left, right *Identity) bool {
	if left == nil && right == nil {
		return true
	}
	if left == nil || right == nil {
		return false
	}

	if len(left.versions) != len(right.versions) {
		return false
	}

	for i, lv := range left.versions {
		rv := right.versions[i]
		if !versionsEqual(lv, rv) {
			return false
		}
	}

	return true
}

// versionsEqual compares two versions, comparing keys by their public key fingerprints
func versionsEqual(left, right *version) bool {
	if left == nil && right == nil {
		return true
	}
	if left == nil || right == nil {
		return false
	}

	// Compare basic fields
	if left.name != right.name || left.email != right.email ||
		left.login != right.login || left.avatarURL != right.avatarURL ||
		left.unixTime != right.unixTime {
		return false
	}

	// Compare times
	if len(left.times) != len(right.times) {
		return false
	}
	for k, tv := range left.times {
		ov, ok := right.times[k]
		if !ok || tv != ov {
			return false
		}
	}

	// Compare keys by fingerprint
	if len(left.keys) != len(right.keys) {
		return false
	}
	for i, k := range left.keys {
		if !keysEqual(k, right.keys[i]) {
			return false
		}
	}

	// Compare nonce
	if len(left.nonce) != len(right.nonce) {
		return false
	}
	for i, n := range left.nonce {
		if n != right.nonce[i] {
			return false
		}
	}

	// Compare metadata
	if len(left.metadata) != len(right.metadata) {
		return false
	}
	for k, vm := range left.metadata {
		om, ok := right.metadata[k]
		if !ok || vm != om {
			return false
		}
	}

	// Don't compare id and commitHash as they're derived fields
	return true
}

// keysEqual compares two keys by their public key fingerprint
func keysEqual(left, right *Key) bool {
	if left == nil && right == nil {
		return true
	}
	if left == nil || right == nil {
		return false
	}
	if left.public == nil && right.public == nil {
		return true
	}
	if left.public == nil || right.public == nil {
		return false
	}
	return bytes.Equal(left.public.Fingerprint[:], right.public.Fingerprint[:])
}

// Test the commit and load of an Identity with multiple versions
func TestIdentityCommitLoad(t *testing.T) {
	repo := makeIdentityTestRepo(t)

	// single version

	identity, err := NewIdentity(repo, "René Descartes", "rene.descartes@example.com")
	require.NoError(t, err)

	idBeforeCommit := identity.Id()

	err = identity.Commit(repo)
	require.NoError(t, err)

	commitsAreSet(t, identity)
	require.NotEmpty(t, identity.Id())
	require.Equal(t, idBeforeCommit, identity.Id())
	require.Equal(t, idBeforeCommit, identity.versions[0].Id())

	loaded, err := ReadLocal(repo, identity.Id())
	require.NoError(t, err)
	commitsAreSet(t, loaded)
	require.True(t, identitiesEqual(identity, loaded), "loaded identity should equal original (comparing by key fingerprint)")

	// multiple versions
	testIdentity := &mockIdentity{name: "René Descartes", email: "rene.descartes@example.com"}
	identity, err = NewIdentityFull(repo, "René Descartes", "rene.descartes@example.com", "", "", []*Key{generatePublicKey(testIdentity)})
	require.NoError(t, err)

	idBeforeCommit = identity.Id()

	err = identity.Mutate(repo, func(orig *Mutator) {
		orig.Keys = []*Key{generatePublicKey(testIdentity)}
	})
	require.NoError(t, err)

	err = identity.Mutate(repo, func(orig *Mutator) {
		orig.Keys = []*Key{generatePublicKey(testIdentity)}
	})
	require.NoError(t, err)

	require.Equal(t, idBeforeCommit, identity.Id())

	err = identity.Commit(repo)
	require.NoError(t, err)

	commitsAreSet(t, identity)
	require.NotEmpty(t, identity.Id())
	require.Equal(t, idBeforeCommit, identity.Id())
	require.Equal(t, idBeforeCommit, identity.versions[0].Id())

	loaded, err = ReadLocal(repo, identity.Id())
	require.NoError(t, err)
	commitsAreSet(t, loaded)
	require.True(t, identitiesEqual(identity, loaded), "loaded identity should equal original (comparing by key fingerprint)")

	// add more version

	err = identity.Mutate(repo, func(orig *Mutator) {
		orig.Email = "rene@descartes.com"
		orig.Keys = []*Key{generatePublicKey(testIdentity)}
	})
	require.NoError(t, err)

	err = identity.Mutate(repo, func(orig *Mutator) {
		orig.Email = "rene@descartes.com"
		orig.Keys = []*Key{generatePublicKey(testIdentity), generatePublicKey(testIdentity)}
	})
	require.NoError(t, err)

	err = identity.Commit(repo)
	require.NoError(t, err)

	commitsAreSet(t, identity)
	require.NotEmpty(t, identity.Id())
	require.Equal(t, idBeforeCommit, identity.Id())
	require.Equal(t, idBeforeCommit, identity.versions[0].Id())

	loaded, err = ReadLocal(repo, identity.Id())
	require.NoError(t, err)
	commitsAreSet(t, loaded)
	require.True(t, identitiesEqual(identity, loaded), "loaded identity should equal original (comparing by key fingerprint)")
}

func TestIdentityMutate(t *testing.T) {
	repo := makeIdentityTestRepo(t)

	identity, err := NewIdentity(repo, "René Descartes", "rene.descartes@example.com")
	require.NoError(t, err)

	require.Len(t, identity.versions, 1)

	err = identity.Mutate(repo, func(orig *Mutator) {
		orig.Email = "rene@descartes.fr"
		orig.Name = "René"
		orig.Login = "rene"
	})
	require.NoError(t, err)

	require.Len(t, identity.versions, 2)
	require.Equal(t, identity.Email(), "rene@descartes.fr")
	require.Equal(t, identity.Name(), "René")
	require.Equal(t, identity.Login(), "rene")
}

func commitsAreSet(t *testing.T, identity *Identity) {
	for _, version := range identity.versions {
		require.NotEmpty(t, version.commitHash)
	}
}

// Test that the correct crypto keys are returned for a given lamport time
func TestIdentity_ValidKeysAtTime(t *testing.T) {
	testIdentity := &mockIdentity{name: "Test User", email: "test@example.com"}
	pubKeyA := generatePublicKey(testIdentity)
	pubKeyB := generatePublicKey(testIdentity)
	pubKeyC := generatePublicKey(testIdentity)
	pubKeyD := generatePublicKey(testIdentity)
	pubKeyE := generatePublicKey(testIdentity)

	identity := Identity{
		versions: []*version{
			{
				times: map[string]lamport.Time{"foo": 100},
				keys:  []*Key{pubKeyA},
			},
			{
				times: map[string]lamport.Time{"foo": 200},
				keys:  []*Key{pubKeyB},
			},
			{
				times: map[string]lamport.Time{"foo": 201},
				keys:  []*Key{pubKeyC},
			},
			{
				times: map[string]lamport.Time{"foo": 201},
				keys:  []*Key{pubKeyD},
			},
			{
				times: map[string]lamport.Time{"foo": 300},
				keys:  []*Key{pubKeyE},
			},
		},
	}

	require.Nil(t, identity.ValidKeysAtTime("foo", 10))
	require.Equal(t, identity.ValidKeysAtTime("foo", 100), []*Key{pubKeyA})
	require.Equal(t, identity.ValidKeysAtTime("foo", 140), []*Key{pubKeyA})
	require.Equal(t, identity.ValidKeysAtTime("foo", 200), []*Key{pubKeyB})
	require.Equal(t, identity.ValidKeysAtTime("foo", 201), []*Key{pubKeyD})
	require.Equal(t, identity.ValidKeysAtTime("foo", 202), []*Key{pubKeyD})
	require.Equal(t, identity.ValidKeysAtTime("foo", 300), []*Key{pubKeyE})
	require.Equal(t, identity.ValidKeysAtTime("foo", 3000), []*Key{pubKeyE})
}

// Test the immutable or mutable metadata search
func TestMetadata(t *testing.T) {
	repo := makeIdentityTestRepo(t)

	identity, err := NewIdentity(repo, "René Descartes", "rene.descartes@example.com")
	require.NoError(t, err)

	identity.SetMetadata("key1", "value1")
	assertHasKeyValue(t, identity.ImmutableMetadata(), "key1", "value1")
	assertHasKeyValue(t, identity.MutableMetadata(), "key1", "value1")

	err = identity.Commit(repo)
	require.NoError(t, err)

	assertHasKeyValue(t, identity.ImmutableMetadata(), "key1", "value1")
	assertHasKeyValue(t, identity.MutableMetadata(), "key1", "value1")

	// try override
	err = identity.Mutate(repo, func(orig *Mutator) {
		orig.Email = "rene@descartes.fr"
	})
	require.NoError(t, err)

	identity.SetMetadata("key1", "value2")
	assertHasKeyValue(t, identity.ImmutableMetadata(), "key1", "value1")
	assertHasKeyValue(t, identity.MutableMetadata(), "key1", "value2")

	err = identity.Commit(repo)
	require.NoError(t, err)

	// reload
	loaded, err := ReadLocal(repo, identity.Id())
	require.NoError(t, err)

	assertHasKeyValue(t, loaded.ImmutableMetadata(), "key1", "value1")
	assertHasKeyValue(t, loaded.MutableMetadata(), "key1", "value2")

	// set metadata after commit
	versionCount := len(identity.versions)
	identity.SetMetadata("foo", "bar")
	require.True(t, identity.NeedCommit())
	require.Len(t, identity.versions, versionCount+1)

	err = identity.Commit(repo)
	require.NoError(t, err)
	require.Len(t, identity.versions, versionCount+1)
}

func assertHasKeyValue(t *testing.T, metadata map[string]string, key, value string) {
	val, ok := metadata[key]
	require.True(t, ok)
	require.Equal(t, val, value)
}

func TestJSON(t *testing.T) {
	repo := makeIdentityTestRepo(t)

	identity, err := NewIdentity(repo, "René Descartes", "rene.descartes@example.com")
	require.NoError(t, err)

	// commit to make sure we have an Id
	err = identity.Commit(repo)
	require.NoError(t, err)
	require.NotEmpty(t, identity.Id())

	// serialize
	data, err := json.Marshal(identity)
	require.NoError(t, err)

	// deserialize, got a IdentityStub with the same id
	var i Interface
	i, err = UnmarshalJSON(data)
	require.NoError(t, err)
	require.Equal(t, identity.Id(), i.Id())

	// make sure we can load the identity properly
	i, err = ReadLocal(repo, i.Id())
	require.NoError(t, err)
}

func TestIdentityRemove(t *testing.T) {
	repo := repository.CreateGoGitTestRepo(t, false)
	remoteA := repository.CreateGoGitTestRepo(t, true)
	remoteB := repository.CreateGoGitTestRepo(t, true)

	err := repo.AddRemote("remoteA", remoteA.GetLocalRemote())
	require.NoError(t, err)

	err = repo.AddRemote("remoteB", remoteB.GetLocalRemote())
	require.NoError(t, err)

	// generate an identity for testing
	rene, err := NewIdentity(repo, "René Descartes", "rene@descartes.fr")
	require.NoError(t, err)

	err = rene.Commit(repo)
	require.NoError(t, err)

	_, err = Push(repo, "remoteA")
	require.NoError(t, err)

	_, err = Push(repo, "remoteB")
	require.NoError(t, err)

	_, err = Fetch(repo, "remoteA")
	require.NoError(t, err)

	_, err = Fetch(repo, "remoteB")
	require.NoError(t, err)

	err = Remove(repo, rene.Id())
	require.NoError(t, err)

	_, err = ReadLocal(repo, rene.Id())
	require.ErrorAs(t, entity.ErrNotFound{}, err)

	_, err = ReadRemote(repo, "remoteA", string(rene.Id()))
	require.ErrorAs(t, entity.ErrNotFound{}, err)

	_, err = ReadRemote(repo, "remoteB", string(rene.Id()))
	require.ErrorAs(t, entity.ErrNotFound{}, err)

	ids, err := ListLocalIds(repo)
	require.NoError(t, err)
	require.Len(t, ids, 0)
}
