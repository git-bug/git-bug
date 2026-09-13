package identity

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/repository"
	"github.com/git-bug/git-bug/util/lamport"
)

// nullSigners nils out the signer field on all keys so that identities built
// with in-process test signers can be compared with identities loaded from git
// (which have no signer set).
func nullSigners(id *Identity) {
	for _, v := range id.versions {
		for _, k := range v.keys {
			k.signer = nil
		}
	}
}

// injectSigners re-attaches in-process signers to matching keys in a loaded identity.
// This simulates the production path where the user's key is in the SSH agent:
// a key loaded from git has a nil signer, and Signer() falls through to the agent.
func injectSigners(id *Identity, sources ...*Key) {
	for _, v := range id.versions {
		for _, k := range v.keys {
			for _, src := range sources {
				if k.publicKeyMultibase == src.publicKeyMultibase && src.signer != nil {
					k.signer = src.signer
				}
			}
		}
	}
}

// Test the commit and load of an Identity without keys (unprotected).
func TestIdentityCommitLoad(t *testing.T) {
	repo := makeIdentityTestRepo(t)

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
	require.Equal(t, identity, loaded)
	require.False(t, identity.IsProtected())
}

// Test the commit and load of an Identity with signing keys (protected).
func TestIdentityCommitLoadWithKeys(t *testing.T) {
	repo := makeIdentityTestRepo(t)

	k1 := newTestSigningKey(t)
	k2 := newTestSigningKey(t)
	k3 := newTestSigningKey(t)
	k4 := newTestSigningKey(t)
	k5 := newTestSigningKey(t)

	identity, err := NewIdentityFull(repo, "René Descartes", "rene.descartes@example.com", "", "", []*Key{k1})
	require.NoError(t, err)

	idBeforeCommit := identity.Id()

	require.True(t, identity.IsProtected()) // protected as soon as keys exist, before commit

	// v2 and v3 queued up before first commit
	err = identity.Mutate(repo, func(orig *Mutator) { orig.Keys = []*Key{k2} })
	require.NoError(t, err)
	err = identity.Mutate(repo, func(orig *Mutator) { orig.Keys = []*Key{k3} })
	require.NoError(t, err)

	err = identity.Commit(repo)
	require.NoError(t, err)
	commitsAreSet(t, identity)
	require.Equal(t, idBeforeCommit, identity.Id())

	// add more versions after an already-committed chain
	err = identity.Mutate(repo, func(orig *Mutator) {
		orig.Email = "rene@descartes.com"
		orig.Keys = []*Key{k4}
	})
	require.NoError(t, err)
	err = identity.Mutate(repo, func(orig *Mutator) {
		orig.Email = "rene@descartes.com"
		orig.Keys = []*Key{k4, k5}
	})
	require.NoError(t, err)

	err = identity.Commit(repo)
	require.NoError(t, err)
	commitsAreSet(t, identity)

	// load and compare — null signers first since loaded keys have no signer set
	nullSigners(identity)
	loaded, err := ReadLocal(repo, identity.Id())
	require.NoError(t, err)
	commitsAreSet(t, loaded)
	require.Equal(t, identity, loaded)
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
	pubKeyA := newTestKey(t)
	pubKeyB := newTestKey(t)
	pubKeyC := newTestKey(t)
	pubKeyD := newTestKey(t)
	pubKeyE := newTestKey(t)

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

// TestIdentityMergeProtectedAccepts verifies that Merge accepts a properly signed
// version from a remote when the identity is protected.
func TestIdentityMergeProtectedAccepts(t *testing.T) {
	repoA := repository.CreateGoGitTestRepo(t, false)
	repoB := repository.CreateGoGitTestRepo(t, false)

	err := repoA.AddRemote("B", repoB.GetLocalRemote())
	require.NoError(t, err)
	err = repoB.AddRemote("A", repoA.GetLocalRemote())
	require.NoError(t, err)

	k1 := newTestSigningKey(t)

	id, err := NewIdentityFull(repoA, "Alice", "alice@example.com", "", "", []*Key{k1})
	require.NoError(t, err)
	err = id.Commit(repoA)
	require.NoError(t, err)
	require.True(t, id.IsProtected())

	_, err = Push(repoA, "B")
	require.NoError(t, err)
	_, err = Fetch(repoB, "A")
	require.NoError(t, err)

	idB, err := ReadLocal(repoB, id.Id())
	require.NoError(t, err)
	// Re-inject signer so idB can commit: mirrors what the SSH agent provides in production.
	injectSigners(idB, k1)

	k2 := newTestSigningKey(t)
	err = idB.Mutate(repoB, func(m *Mutator) { m.Keys = []*Key{k2} })
	require.NoError(t, err)
	err = idB.Commit(repoB)
	require.NoError(t, err)

	_, err = Fetch(repoA, "B")
	require.NoError(t, err)
	remoteId, err := ReadRemote(repoA, "B", string(id.Id()))
	require.NoError(t, err)
	modified, err := id.Merge(repoA, remoteId)
	require.NoError(t, err)
	require.True(t, modified)
}

// TestIdentityMergeProtectedRejects verifies that Merge rejects an unsigned version
// appended to a protected identity. The tampered version is injected directly via
// low-level repo calls since Commit() now blocks unsigned versions on protected identities.
func TestIdentityMergeProtectedRejects(t *testing.T) {
	repo := makeIdentityTestRepo(t)

	k1 := newTestSigningKey(t)

	id, err := NewIdentityFull(repo, "Alice", "alice@example.com", "", "", []*Key{k1})
	require.NoError(t, err)
	err = id.Commit(repo)
	require.NoError(t, err)
	require.True(t, id.IsProtected())

	// Build a tampered version: a second version blob committed WITHOUT a signature.
	// Bypass the normal checks and write directly to git.
	v2, err := newVersion(repo, "Eve", "evil@example.com", "", "", nil)
	require.NoError(t, err)
	blobHash, err := v2.Write(repo)
	require.NoError(t, err)
	treeHash, err := repo.StoreTree([]repository.TreeEntry{
		{ObjectType: repository.Blob, Hash: blobHash, Name: versionEntryName},
	})
	require.NoError(t, err)
	unsignedHash, err := repo.StoreCommit(treeHash, id.versions[0].commitHash)
	require.NoError(t, err)
	v2.commitHash = unsignedHash

	// Construct a remote Identity that looks like it appended this unsigned version.
	tampered := &Identity{versions: append(id.versions[:len(id.versions):len(id.versions)], v2)}

	_, err = id.Merge(repo, tampered)
	require.Error(t, err, "unsigned version appended to protected identity must be rejected")
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
