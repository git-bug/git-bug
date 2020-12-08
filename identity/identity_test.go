package identity

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/repository"
)

// Test the commit and load of an Identity with multiple versions
func TestIdentityCommitLoad(t *testing.T) {
	mockRepo := repository.NewMockRepoForTest()

	// single version

	identity := &Identity{
		id: entity.UnsetId,
		versions: []*Version{
			{
				name:  "René Descartes",
				email: "rene.descartes@example.com",
			},
		},
	}

	err := identity.Commit(mockRepo)

	require.NoError(t, err)
	require.NotEmpty(t, identity.id)

	loaded, err := ReadLocal(mockRepo, identity.id)
	require.NoError(t, err)
	commitsAreSet(t, loaded)
	require.Equal(t, identity, loaded)

	// multiple version

	identity = &Identity{
		id: entity.UnsetId,
		versions: []*Version{
			{
				time:  100,
				name:  "René Descartes",
				email: "rene.descartes@example.com",
				keys: []*Key{
					{PubKey: "pubkeyA"},
				},
			},
			{
				time:  200,
				name:  "René Descartes",
				email: "rene.descartes@example.com",
				keys: []*Key{
					{PubKey: "pubkeyB"},
				},
			},
			{
				time:  201,
				name:  "René Descartes",
				email: "rene.descartes@example.com",
				keys: []*Key{
					{PubKey: "pubkeyC"},
				},
			},
		},
	}

	err = identity.Commit(mockRepo)

	require.NoError(t, err)
	require.NotEmpty(t, identity.id)

	loaded, err = ReadLocal(mockRepo, identity.id)
	require.NoError(t, err)
	commitsAreSet(t, loaded)
	require.Equal(t, identity, loaded)

	// add more version

	identity.addVersionForTest(&Version{
		time:  201,
		name:  "René Descartes",
		email: "rene.descartes@example.com",
		keys: []*Key{
			{PubKey: "pubkeyD"},
		},
	})

	identity.addVersionForTest(&Version{
		time:  300,
		name:  "René Descartes",
		email: "rene.descartes@example.com",
		keys: []*Key{
			{PubKey: "pubkeyE"},
		},
	})

	err = identity.Commit(mockRepo)

	require.NoError(t, err)
	require.NotEmpty(t, identity.id)

	loaded, err = ReadLocal(mockRepo, identity.id)
	require.NoError(t, err)
	commitsAreSet(t, loaded)
	require.Equal(t, identity, loaded)
}

func TestIdentityMutate(t *testing.T) {
	identity := NewIdentity("René Descartes", "rene.descartes@example.com")

	require.Len(t, identity.versions, 1)

	identity.Mutate(func(orig Mutator) Mutator {
		orig.Email = "rene@descartes.fr"
		orig.Name = "René"
		orig.Login = "rene"
		return orig
	})

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
	identity := Identity{
		id: entity.UnsetId,
		versions: []*Version{
			{
				time:  100,
				name:  "René Descartes",
				email: "rene.descartes@example.com",
				keys: []*Key{
					{PubKey: "pubkeyA"},
				},
			},
			{
				time:  200,
				name:  "René Descartes",
				email: "rene.descartes@example.com",
				keys: []*Key{
					{PubKey: "pubkeyB"},
				},
			},
			{
				time:  201,
				name:  "René Descartes",
				email: "rene.descartes@example.com",
				keys: []*Key{
					{PubKey: "pubkeyC"},
				},
			},
			{
				time:  201,
				name:  "René Descartes",
				email: "rene.descartes@example.com",
				keys: []*Key{
					{PubKey: "pubkeyD"},
				},
			},
			{
				time:  300,
				name:  "René Descartes",
				email: "rene.descartes@example.com",
				keys: []*Key{
					{PubKey: "pubkeyE"},
				},
			},
		},
	}

	require.Nil(t, identity.ValidKeysAtTime(10))
	require.Equal(t, identity.ValidKeysAtTime(100), []*Key{{PubKey: "pubkeyA"}})
	require.Equal(t, identity.ValidKeysAtTime(140), []*Key{{PubKey: "pubkeyA"}})
	require.Equal(t, identity.ValidKeysAtTime(200), []*Key{{PubKey: "pubkeyB"}})
	require.Equal(t, identity.ValidKeysAtTime(201), []*Key{{PubKey: "pubkeyD"}})
	require.Equal(t, identity.ValidKeysAtTime(202), []*Key{{PubKey: "pubkeyD"}})
	require.Equal(t, identity.ValidKeysAtTime(300), []*Key{{PubKey: "pubkeyE"}})
	require.Equal(t, identity.ValidKeysAtTime(3000), []*Key{{PubKey: "pubkeyE"}})
}

// Test the immutable or mutable metadata search
func TestMetadata(t *testing.T) {
	mockRepo := repository.NewMockRepoForTest()

	identity := NewIdentity("René Descartes", "rene.descartes@example.com")

	identity.SetMetadata("key1", "value1")
	assertHasKeyValue(t, identity.ImmutableMetadata(), "key1", "value1")
	assertHasKeyValue(t, identity.MutableMetadata(), "key1", "value1")

	err := identity.Commit(mockRepo)
	require.NoError(t, err)

	assertHasKeyValue(t, identity.ImmutableMetadata(), "key1", "value1")
	assertHasKeyValue(t, identity.MutableMetadata(), "key1", "value1")

	// try override
	identity.addVersionForTest(&Version{
		name:  "René Descartes",
		email: "rene.descartes@example.com",
	})

	identity.SetMetadata("key1", "value2")
	assertHasKeyValue(t, identity.ImmutableMetadata(), "key1", "value1")
	assertHasKeyValue(t, identity.MutableMetadata(), "key1", "value2")

	err = identity.Commit(mockRepo)
	require.NoError(t, err)

	// reload
	loaded, err := ReadLocal(mockRepo, identity.id)
	require.NoError(t, err)

	assertHasKeyValue(t, loaded.ImmutableMetadata(), "key1", "value1")
	assertHasKeyValue(t, loaded.MutableMetadata(), "key1", "value2")
}

func assertHasKeyValue(t *testing.T, metadata map[string]string, key, value string) {
	val, ok := metadata[key]
	require.True(t, ok)
	require.Equal(t, val, value)
}

func TestJSON(t *testing.T) {
	mockRepo := repository.NewMockRepoForTest()

	identity := &Identity{
		id: entity.UnsetId,
		versions: []*Version{
			{
				name:  "René Descartes",
				email: "rene.descartes@example.com",
			},
		},
	}

	// commit to make sure we have an Id
	err := identity.Commit(mockRepo)
	require.NoError(t, err)
	require.NotEmpty(t, identity.id)

	// serialize
	data, err := json.Marshal(identity)
	require.NoError(t, err)

	// deserialize, got a IdentityStub with the same id
	var i Interface
	i, err = UnmarshalJSON(data)
	require.NoError(t, err)
	require.Equal(t, identity.id, i.Id())

	// make sure we can load the identity properly
	i, err = ReadLocal(mockRepo, i.Id())
	require.NoError(t, err)
}

func TestIdentityRemove(t *testing.T) {
	repo := repository.CreateGoGitTestRepo(false)
	remoteA := repository.CreateGoGitTestRepo(true)
	remoteB := repository.CreateGoGitTestRepo(true)
	defer repository.CleanupTestRepos(repo, remoteA, remoteB)

	err := repo.AddRemote("remoteA", remoteA.GetLocalRemote())
	require.NoError(t, err)

	err = repo.AddRemote("remoteB", remoteB.GetLocalRemote())
	require.NoError(t, err)

	// generate an identity for testing
	rene := NewIdentity("René Descartes", "rene@descartes.fr")
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

	err = RemoveIdentity(repo, rene.Id())
	require.NoError(t, err)

	_, err = ReadLocal(repo, rene.Id())
	require.Error(t, ErrIdentityNotExist, err)

	_, err = ReadRemote(repo, "remoteA", string(rene.Id()))
	require.Error(t, ErrIdentityNotExist, err)

	_, err = ReadRemote(repo, "remoteB", string(rene.Id()))
	require.Error(t, ErrIdentityNotExist, err)

	ids, err := ListLocalIds(repo)
	require.NoError(t, err)
	require.Len(t, ids, 0)
}
