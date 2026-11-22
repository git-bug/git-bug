package auth

import (
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

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

func TestListNewestFirst(t *testing.T) {
	repo := repository.NewMockRepo()

	now := time.Now()
	var stored []*Token
	// store out of chronological order on purpose
	for _, age := range []time.Duration{2 * time.Hour, 0, 3 * time.Hour, time.Hour} {
		token := NewToken("gitea", "value")
		token.createTime = now.Add(-age)
		require.NoError(t, Store(repo, token))
		stored = append(stored, token)
	}

	creds, err := List(repo, WithTarget("gitea"))
	require.NoError(t, err)
	require.Len(t, creds, 4)

	// expected order: age 0, 1h, 2h, 3h
	expected := []entity.Id{stored[1].ID(), stored[3].ID(), stored[0].ID(), stored[2].ID()}
	for i, cred := range creds {
		assert.Equal(t, expected[i], cred.ID(), "position %d", i)
	}
}

func TestListNewestFirstSameSecond(t *testing.T) {
	repo := repository.NewMockRepo()

	now := time.Now().Truncate(time.Second)
	var stored []*Token
	// store out of chronological order within the same second
	for _, subSec := range []time.Duration{500 * time.Millisecond, 0, 750 * time.Millisecond, 250 * time.Millisecond} {
		token := NewToken("gitea", "value")
		token.createTime = now.Add(subSec)
		require.NoError(t, Store(repo, token))
		stored = append(stored, token)
	}

	creds, err := List(repo, WithTarget("gitea"))
	require.NoError(t, err)
	require.Len(t, creds, 4)

	// expected order: 750ms, 500ms, 250ms, 0ms
	expected := []entity.Id{stored[2].ID(), stored[0].ID(), stored[3].ID(), stored[1].ID()}
	for i, cred := range creds {
		assert.Equal(t, expected[i], cred.ID(), "position %d", i)
	}
}

func TestListLegacyTimestamp(t *testing.T) {
	repo := repository.NewMockRepo()

	tokenOld := NewToken("gitea", "old-value")
	rawItem := map[string]string{
		keyringKeyKind:       string(tokenOld.Kind()),
		keyringKeyTarget:     tokenOld.Target(),
		keyringKeyCreateTime: "1700000000",
		keyringKeySalt:       base64.StdEncoding.EncodeToString(tokenOld.Salt()),
		keyringKeyTokenValue: tokenOld.Value,
	}
	data, err := json.Marshal(rawItem)
	require.NoError(t, err)
	require.NoError(t, repo.Keyring().Set(repository.Item{
		Key:  keyringKeyPrefix + tokenOld.ID().String(),
		Data: data,
	}))

	tokenNew := NewToken("gitea", "new-value")
	tokenNew.createTime = time.Unix(1700001000, 500000000)
	require.NoError(t, Store(repo, tokenNew))

	creds, err := List(repo, WithTarget("gitea"))
	require.NoError(t, err)
	require.Len(t, creds, 2)
	assert.Equal(t, tokenNew.ID(), creds[0].ID())
	assert.Equal(t, tokenOld.ID(), creds[1].ID())
}

func TestListBaseURLNormalization(t *testing.T) {
	repo := repository.NewMockRepo()

	tokenWithoutSlash := NewToken("gitlab", "value-1")
	tokenWithoutSlash.SetMetadata(MetaKeyBaseURL, "https://gitlab.com")
	require.NoError(t, Store(repo, tokenWithoutSlash))

	tokenWithSlash := NewToken("gitlab", "value-2")
	tokenWithSlash.SetMetadata(MetaKeyBaseURL, "https://gitlab.example.com/")
	require.NoError(t, Store(repo, tokenWithSlash))

	// Query with trailing slash against token without trailing slash
	creds, err := List(repo, WithTarget("gitlab"), WithMeta(MetaKeyBaseURL, "https://gitlab.com/"))
	require.NoError(t, err)
	require.Len(t, creds, 1)
	assert.Equal(t, tokenWithoutSlash.ID(), creds[0].ID())

	// Query without trailing slash against token with trailing slash
	creds, err = List(repo, WithTarget("gitlab"), WithMeta(MetaKeyBaseURL, "https://gitlab.example.com"))
	require.NoError(t, err)
	require.Len(t, creds, 1)
	assert.Equal(t, tokenWithSlash.ID(), creds[0].ID())
}

func TestNormalizeBaseURL(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", ""},
		{"   ", ""},
		{"https://gitlab.com", "https://gitlab.com"},
		{"https://gitlab.com/", "https://gitlab.com"},
		{"https://gitlab.com///", "https://gitlab.com"},
		{"HTTPS://GITLAB.COM/", "https://gitlab.com"},
		{"https://gitlab.example.com/sub/group/", "https://gitlab.example.com/sub/group"},
		{"https://gitlab.example.com/sub/group", "https://gitlab.example.com/sub/group"},
		{"http://jira.local:8080/jira/", "http://jira.local:8080/jira"},
		{"http://jira.local:8080/jira", "http://jira.local:8080/jira"},
		{"jira.example.com/", "jira.example.com"},
		{"jira.example.com", "jira.example.com"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			assert.Equal(t, tc.want, NormalizeBaseURL(tc.input))
		})
	}
}
