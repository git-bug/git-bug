package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/repository"
)

func TestCredential(t *testing.T) {
	repo := repository.NewMockRepo()

	storeToken := func(val string, target string) *Token {
		token := NewToken(target, val)
		err := Store(repo, token)
		require.NoError(t, err)
		return token
	}

	token := storeToken("foobar", "github")

	// Store + Load
	err := Store(repo, token)
	assert.NoError(t, err)

	token2, err := LoadWithId(repo, token.ID())
	assert.NoError(t, err)
	assert.Equal(t, token.createTime.Unix(), token2.CreateTime().Unix())
	token.createTime = token2.CreateTime()
	assert.Equal(t, token, token2)

	prefix := string(token.ID())[:10]

	// LoadWithPrefix
	token3, err := LoadWithPrefix(repo, prefix)
	assert.NoError(t, err)
	assert.Equal(t, token.createTime.Unix(), token3.CreateTime().Unix())
	token.createTime = token3.CreateTime()
	assert.Equal(t, token, token3)

	token4 := storeToken("foo", "gitlab")
	token5 := storeToken("bar", "github")

	// List + options
	creds, err := List(repo, WithTarget("github"))
	assert.NoError(t, err)
	sameIds(t, creds, []Credential{token, token5})

	creds, err = List(repo, WithTarget("gitlab"))
	assert.NoError(t, err)
	sameIds(t, creds, []Credential{token4})

	creds, err = List(repo, WithKind(KindToken))
	assert.NoError(t, err)
	sameIds(t, creds, []Credential{token, token4, token5})

	creds, err = List(repo, WithKind(KindLoginPassword))
	assert.NoError(t, err)
	sameIds(t, creds, []Credential{})

	// Metadata

	token4.SetMetadata("key", "value")
	err = Store(repo, token4)
	assert.NoError(t, err)

	creds, err = List(repo, WithMeta("key", "value"))
	assert.NoError(t, err)
	sameIds(t, creds, []Credential{token4})

	// Exist
	exist := IdExist(repo, token.ID())
	assert.True(t, exist)

	exist = PrefixExist(repo, prefix)
	assert.True(t, exist)

	// Remove
	err = Remove(repo, token.ID())
	assert.NoError(t, err)

	creds, err = List(repo)
	assert.NoError(t, err)
	sameIds(t, creds, []Credential{token4, token5})
}

func sameIds(t *testing.T, a []Credential, b []Credential) {
	t.Helper()

	ids := func(creds []Credential) []entity.Id {
		result := make([]entity.Id, len(creds))
		for i, cred := range creds {
			result[i] = cred.ID()
		}
		return result
	}

	assert.ElementsMatch(t, ids(a), ids(b))
}

func testCredentialSerial(t *testing.T, original Credential) Credential {
	repo := repository.NewMockRepo()

	original.SetMetadata("test", "value")

	assert.NotEmpty(t, original.ID().String())
	assert.NotEmpty(t, original.Salt())
	assert.NoError(t, Store(repo, original))

	loaded, err := LoadWithId(repo, original.ID())
	assert.NoError(t, err)

	assert.Equal(t, original.ID(), loaded.ID())
	assert.Equal(t, original.Kind(), loaded.Kind())
	assert.Equal(t, original.Target(), loaded.Target())
	assert.Equal(t, original.CreateTime().Unix(), loaded.CreateTime().Unix())
	assert.Equal(t, original.Salt(), loaded.Salt())
	assert.Equal(t, original.Metadata(), loaded.Metadata())

	return loaded
}
