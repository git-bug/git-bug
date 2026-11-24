package identity

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/repository"
	"github.com/git-bug/git-bug/util/lamport"
)

func makeIdentityTestRepo(t *testing.T) repository.Repo {
	repo := repository.NewMockRepo()

	_, err := repo.GetOrCreateClock("foo", 42)
	require.NoError(t, err)

	_, err = repo.GetOrCreateClock("bar", 34)
	require.NoError(t, err)

	return repo
}

func TestVersionJSON(t *testing.T) {
	repo := makeIdentityTestRepo(t)

	keys := []*Key{
		generatePublicKey(),
		generatePublicKey(),
	}

	before, err := newVersion(repo, "name", "email", "login", "https://example.com/avatar.png", keys)
	require.NoError(t, err)

	before.SetMetadata("key1", "value1")
	before.SetMetadata("key2", "value2")

	expected := &version{
		id:        entity.UnsetId,
		name:      "name",
		email:     "email",
		login:     "login",
		avatarURL: "https://example.com/avatar.png",
		unixTime:  time.Now().Unix(),
		times: map[string]lamport.Time{
			"foo": 42,
			"bar": 34,
		},
		keys:  keys,
		nonce: before.nonce,
		metadata: map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
	}

	require.Equal(t, expected, before)

	data, err := json.Marshal(before)
	assert.NoError(t, err)

	var after version
	err = json.Unmarshal(data, &after)
	assert.NoError(t, err)

	// make sure we now have an Id
	expected.Id()

	assert.Equal(t, expected, &after)
}

func TestVersionAvatarURL(t *testing.T) {
	// a stored avatar URL that isn't an HTTPS URL reads as absent, and the
	// version stays valid
	cases := []struct {
		name   string
		url    string
		readAs string
	}{
		{"an HTTPS URL", "https://example.com/avatar.png", "https://example.com/avatar.png"},
		{"a port, as a self-hosted GitLab gave", "https://gitlab.example.org:8929/avatar.png", ""},
		{"http, as a self-hosted GitLab gave", "http://gitlab.example.org/avatar.png", ""},
		{"a path, as user new accepted", "/me.png", ""},
		{"a javascript URL", "javascript:alert(1)", ""},
		{"an IP address", "https://127.0.0.1/avatar.png", ""},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			raw, err := json.Marshal(map[string]any{
				"version":    2,
				"times":      map[string]any{},
				"unix_time":  1609459200,
				"name":       "Alice",
				"avatar_url": c.url,
				"nonce":      "AAAAAAAAAAAAAAAAAAAAAAAAAAA=",
			})
			require.NoError(t, err)
			var v version
			require.NoError(t, json.Unmarshal(raw, &v))
			require.Equal(t, c.readAs, v.avatarURL)
			require.NoError(t, v.Validate())
		})
	}

	t.Run("a writer never writes one", func(t *testing.T) {
		v, err := newVersion(makeIdentityTestRepo(t), "Alice", "", "", "http://example.com/avatar.png", nil)
		require.NoError(t, err)
		require.ErrorContains(t, v.Validate(), "avatarUrl is not an HTTPS URL")
	})
}
