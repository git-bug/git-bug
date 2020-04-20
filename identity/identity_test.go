package identity

import (
	"encoding/json"
	"testing"

	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/repository"
	"github.com/stretchr/testify/assert"
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

	assert.Nil(t, err)
	assert.NotEmpty(t, identity.id)

	loaded, err := ReadLocal(mockRepo, identity.id)
	assert.Nil(t, err)
	commitsAreSet(t, loaded)
	assert.Equal(t, identity, loaded)

	// multiple version

	identity = &Identity{
		id: entity.UnsetId,
		versions: []*Version{
			{
				time:  100,
				name:  "René Descartes",
				email: "rene.descartes@example.com",
				keys: []*Key{
					{armoredPublicKey: repository.CreatePubkey(t)},
				},
			},
			{
				time:  200,
				name:  "René Descartes",
				email: "rene.descartes@example.com",
				keys: []*Key{
					{armoredPublicKey: repository.CreatePubkey(t)},
				},
			},
			{
				time:  201,
				name:  "René Descartes",
				email: "rene.descartes@example.com",
				keys: []*Key{
					{armoredPublicKey: repository.CreatePubkey(t)},
				},
			},
		},
	}

	err = identity.Commit(mockRepo)

	assert.Nil(t, err)
	assert.NotEmpty(t, identity.id)

	loaded, err = ReadLocal(mockRepo, identity.id)
	assert.Nil(t, err)
	commitsAreSet(t, loaded)
	loadKeys(loaded)
	assert.Equal(t, identity, loaded)

	// add more version

	identity.addVersionForTest(&Version{
		time:  201,
		name:  "René Descartes",
		email: "rene.descartes@example.com",
		keys: []*Key{
			{armoredPublicKey: repository.CreatePubkey(t)},
		},
	})

	identity.addVersionForTest(&Version{
		time:  300,
		name:  "René Descartes",
		email: "rene.descartes@example.com",
		keys: []*Key{
			{armoredPublicKey: repository.CreatePubkey(t)},
		},
	})

	err = identity.Commit(mockRepo)

	assert.Nil(t, err)
	assert.NotEmpty(t, identity.id)

	loaded, err = ReadLocal(mockRepo, identity.id)
	assert.Nil(t, err)
	commitsAreSet(t, loaded)
	loadKeys(loaded)
	assert.Equal(t, identity, loaded)
}

func TestIdentityMutate(t *testing.T) {
	identity := NewIdentity("René Descartes", "rene.descartes@example.com")

	assert.Len(t, identity.versions, 1)

	identity.Mutate(func(orig Mutator) Mutator {
		orig.Email = "rene@descartes.fr"
		orig.Name = "René"
		orig.Login = "rene"
		return orig
	})

	assert.Len(t, identity.versions, 2)
	assert.Equal(t, identity.Email(), "rene@descartes.fr")
	assert.Equal(t, identity.Name(), "René")
	assert.Equal(t, identity.Login(), "rene")
}

func loadKeys(identity *Identity) {
	for _, v := range identity.versions {
		for _, k := range v.keys {
			k.PublicKey()
		}
	}
}

func commitsAreSet(t *testing.T, identity *Identity) {
	for _, version := range identity.versions {
		assert.NotEmpty(t, version.commitHash)
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
					{armoredPublicKey: "pubkeyA"},
				},
			},
			{
				time:  200,
				name:  "René Descartes",
				email: "rene.descartes@example.com",
				keys: []*Key{
					{armoredPublicKey: "pubkeyB"},
				},
			},
			{
				time:  201,
				name:  "René Descartes",
				email: "rene.descartes@example.com",
				keys: []*Key{
					{armoredPublicKey: "pubkeyC"},
				},
			},
			{
				time:  201,
				name:  "René Descartes",
				email: "rene.descartes@example.com",
				keys: []*Key{
					{armoredPublicKey: "pubkeyD"},
				},
			},
			{
				time:  300,
				name:  "René Descartes",
				email: "rene.descartes@example.com",
				keys: []*Key{
					{armoredPublicKey: "pubkeyE"},
				},
			},
		},
	}

	assert.Nil(t, identity.ValidKeysAtTime(10))
	assert.Equal(t, identity.ValidKeysAtTime(100), []*Key{{armoredPublicKey: "pubkeyA"}})
	assert.Equal(t, identity.ValidKeysAtTime(140), []*Key{{armoredPublicKey: "pubkeyA"}})
	assert.Equal(t, identity.ValidKeysAtTime(200), []*Key{{armoredPublicKey: "pubkeyB"}})
	assert.Equal(t, identity.ValidKeysAtTime(201), []*Key{{armoredPublicKey: "pubkeyD"}})
	assert.Equal(t, identity.ValidKeysAtTime(202), []*Key{{armoredPublicKey: "pubkeyD"}})
	assert.Equal(t, identity.ValidKeysAtTime(300), []*Key{{armoredPublicKey: "pubkeyE"}})
	assert.Equal(t, identity.ValidKeysAtTime(3000), []*Key{{armoredPublicKey: "pubkeyE"}})
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
	identity.addVersionForTest(&Version{
		name:  "René Descartes",
		email: "rene.descartes@example.com",
	})

	identity.SetMetadata("key1", "value2")
	assertHasKeyValue(t, identity.ImmutableMetadata(), "key1", "value1")
	assertHasKeyValue(t, identity.MutableMetadata(), "key1", "value2")

	err = identity.Commit(mockRepo)
	assert.NoError(t, err)

	// reload
	loaded, err := ReadLocal(mockRepo, identity.id)
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
	i, err = ReadLocal(mockRepo, i.Id())
	assert.NoError(t, err)
}
