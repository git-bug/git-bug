package identity

import (
	"encoding/json"
	"testing"

	"github.com/MichaelMure/git-bug/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test the commit and load of an Identity with multiple versions
func TestIdentityCommitLoad(t *testing.T) {
	mockRepo := repository.NewMockRepoForTest()

	// single version

	identity := &Identity{
		Versions: []*Version{
			{
				Name:  "René Descartes",
				Email: "rene.descartes@example.com",
			},
		},
	}

	err := identity.Commit(mockRepo)

	assert.Nil(t, err)
	assert.NotEmpty(t, identity.id)

	loaded, err := Read(mockRepo, identity.id)
	assert.Nil(t, err)
	commitsAreSet(t, loaded)
	equivalentIdentity(t, identity, loaded)

	// multiple version

	identity = &Identity{
		Versions: []*Version{
			{
				Time:  100,
				Name:  "René Descartes",
				Email: "rene.descartes@example.com",
				Keys: []Key{
					{PubKey: "pubkeyA"},
				},
			},
			{
				Time:  200,
				Name:  "René Descartes",
				Email: "rene.descartes@example.com",
				Keys: []Key{
					{PubKey: "pubkeyB"},
				},
			},
			{
				Time:  201,
				Name:  "René Descartes",
				Email: "rene.descartes@example.com",
				Keys: []Key{
					{PubKey: "pubkeyC"},
				},
			},
		},
	}

	err = identity.Commit(mockRepo)

	assert.Nil(t, err)
	assert.NotEmpty(t, identity.id)

	loaded, err = Read(mockRepo, identity.id)
	assert.Nil(t, err)
	commitsAreSet(t, loaded)
	equivalentIdentity(t, identity, loaded)

	// add more version

	identity.AddVersion(&Version{
		Time:  201,
		Name:  "René Descartes",
		Email: "rene.descartes@example.com",
		Keys: []Key{
			{PubKey: "pubkeyD"},
		},
	})

	identity.AddVersion(&Version{
		Time:  300,
		Name:  "René Descartes",
		Email: "rene.descartes@example.com",
		Keys: []Key{
			{PubKey: "pubkeyE"},
		},
	})

	err = identity.Commit(mockRepo)

	assert.Nil(t, err)
	assert.NotEmpty(t, identity.id)

	loaded, err = Read(mockRepo, identity.id)
	assert.Nil(t, err)
	commitsAreSet(t, loaded)
	equivalentIdentity(t, identity, loaded)
}

func commitsAreSet(t *testing.T, identity *Identity) {
	for _, version := range identity.Versions {
		assert.NotEmpty(t, version.commitHash)
	}
}

func equivalentIdentity(t *testing.T, expected, actual *Identity) {
	require.Equal(t, len(expected.Versions), len(actual.Versions))

	for i, version := range expected.Versions {
		actual.Versions[i].commitHash = version.commitHash
	}

	assert.Equal(t, expected, actual)
}

// Test that the correct crypto keys are returned for a given lamport time
func TestIdentity_ValidKeysAtTime(t *testing.T) {
	identity := Identity{
		Versions: []*Version{
			{
				Time:  100,
				Name:  "René Descartes",
				Email: "rene.descartes@example.com",
				Keys: []Key{
					{PubKey: "pubkeyA"},
				},
			},
			{
				Time:  200,
				Name:  "René Descartes",
				Email: "rene.descartes@example.com",
				Keys: []Key{
					{PubKey: "pubkeyB"},
				},
			},
			{
				Time:  201,
				Name:  "René Descartes",
				Email: "rene.descartes@example.com",
				Keys: []Key{
					{PubKey: "pubkeyC"},
				},
			},
			{
				Time:  201,
				Name:  "René Descartes",
				Email: "rene.descartes@example.com",
				Keys: []Key{
					{PubKey: "pubkeyD"},
				},
			},
			{
				Time:  300,
				Name:  "René Descartes",
				Email: "rene.descartes@example.com",
				Keys: []Key{
					{PubKey: "pubkeyE"},
				},
			},
		},
	}

	assert.Nil(t, identity.ValidKeysAtTime(10))
	assert.Equal(t, identity.ValidKeysAtTime(100), []Key{{PubKey: "pubkeyA"}})
	assert.Equal(t, identity.ValidKeysAtTime(140), []Key{{PubKey: "pubkeyA"}})
	assert.Equal(t, identity.ValidKeysAtTime(200), []Key{{PubKey: "pubkeyB"}})
	assert.Equal(t, identity.ValidKeysAtTime(201), []Key{{PubKey: "pubkeyD"}})
	assert.Equal(t, identity.ValidKeysAtTime(202), []Key{{PubKey: "pubkeyD"}})
	assert.Equal(t, identity.ValidKeysAtTime(300), []Key{{PubKey: "pubkeyE"}})
	assert.Equal(t, identity.ValidKeysAtTime(3000), []Key{{PubKey: "pubkeyE"}})
}

// Test the immutable or mutable metadata search
func TestMetadata(t *testing.T) {
	mockRepo := repository.NewMockRepoForTest()

	identity := NewIdentity("René Descartes", "rene.descartes@example.com")

	identity.SetMetadata("key1", "value1")
	assertHasKeyValue(t, identity.ImmutableMetadata(), "key1", "value1")
	assertHasKeyValue(t, identity.MutableMetadata(), "key1", "value1")

	err := identity.Commit(mockRepo)
	assert.NoError(t, err)

	assertHasKeyValue(t, identity.ImmutableMetadata(), "key1", "value1")
	assertHasKeyValue(t, identity.MutableMetadata(), "key1", "value1")

	// try override
	identity.AddVersion(&Version{
		Name:  "René Descartes",
		Email: "rene.descartes@example.com",
	})

	identity.SetMetadata("key1", "value2")
	assertHasKeyValue(t, identity.ImmutableMetadata(), "key1", "value1")
	assertHasKeyValue(t, identity.MutableMetadata(), "key1", "value2")

	err = identity.Commit(mockRepo)
	assert.NoError(t, err)

	// reload
	loaded, err := Read(mockRepo, identity.id)
	assert.Nil(t, err)

	assertHasKeyValue(t, loaded.ImmutableMetadata(), "key1", "value1")
	assertHasKeyValue(t, loaded.MutableMetadata(), "key1", "value2")
}

func assertHasKeyValue(t *testing.T, metadata map[string]string, key, value string) {
	val, ok := metadata[key]
	assert.True(t, ok)
	assert.Equal(t, val, value)
}

func TestJSON(t *testing.T) {
	mockRepo := repository.NewMockRepoForTest()

	identity := &Identity{
		Versions: []*Version{
			{
				Name:  "René Descartes",
				Email: "rene.descartes@example.com",
			},
		},
	}

	// commit to make sure we have an ID
	err := identity.Commit(mockRepo)
	assert.Nil(t, err)
	assert.NotEmpty(t, identity.id)

	// serialize
	data, err := json.Marshal(identity)
	assert.NoError(t, err)

	// deserialize, got a IdentityStub with the same id
	var i Interface
	i, err = UnmarshalJSON(data)
	assert.NoError(t, err)
	assert.Equal(t, identity.id, i.Id())

	// make sure we can load the identity properly
	i, err = Read(mockRepo, i.Id())
	assert.NoError(t, err)
}
