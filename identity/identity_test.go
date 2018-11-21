package identity

import (
	"testing"

	"github.com/MichaelMure/git-bug/repository"
	"github.com/stretchr/testify/assert"
)

func TestIdentityCommit(t *testing.T) {
	mockRepo := repository.NewMockRepoForTest()

	// single version

	identity := Identity{
		Versions: []Version{
			{
				Name:  "René Descartes",
				Email: "rene.descartes@example.com",
			},
		},
	}

	err := identity.Commit(mockRepo)

	assert.Nil(t, err)
	assert.NotEmpty(t, identity.id)

	// multiple version

	identity = Identity{
		Versions: []Version{
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

	// add more version

	identity.AddVersion(Version{
		Time:  201,
		Name:  "René Descartes",
		Email: "rene.descartes@example.com",
		Keys: []Key{
			{PubKey: "pubkeyD"},
		},
	})

	identity.AddVersion(Version{
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
}

func TestIdentity_ValidKeysAtTime(t *testing.T) {
	identity := Identity{
		Versions: []Version{
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
