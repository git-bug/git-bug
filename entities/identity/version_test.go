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

func makeIdentityTestRepo(t *testing.T) repository.ClockedRepo {
	repo := repository.NewMockRepo()

	clock1, err := repo.GetOrCreateClock("foo")
	require.NoError(t, err)
	err = clock1.Witness(42)
	require.NoError(t, err)

	clock2, err := repo.GetOrCreateClock("bar")
	require.NoError(t, err)
	err = clock2.Witness(34)
	require.NoError(t, err)

	return repo
}

func TestVersionJSON(t *testing.T) {
	repo := makeIdentityTestRepo(t)

	keys := []*Key{
		newTestKey(t),
		newTestKey(t),
	}

	before, err := newVersion(repo, "name", "email", "login", "avatarUrl", keys)
	require.NoError(t, err)

	before.SetMetadata("key1", "value1")
	before.SetMetadata("key2", "value2")

	expected := &version{
		id:        entity.UnsetId,
		name:      "name",
		email:     "email",
		login:     "login",
		avatarURL: "avatarUrl",
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

// Format v2 identities still load, but their pub_keys are ignored: the v2 key
// mechanism was never functional and no identity in the wild has keys.
func TestVersionJSONFormatV2KeysIgnored(t *testing.T) {
	v2JSON := []byte(`{
		"version": 2,
		"times": {"bugs-create": 1},
		"unix_time": 1609459200,
		"name": "Alice",
		"email": "alice@example.com",
		"pub_keys": ["-----BEGIN PGP PUBLIC KEY BLOCK-----\n...\n-----END PGP PUBLIC KEY BLOCK-----"],
		"nonce": "AAAAAAAAAAAAAAAAAAAAAAAAAAAA"
	}`)

	var v version
	require.NoError(t, json.Unmarshal(v2JSON, &v))
	require.Equal(t, "Alice", v.name)
	require.Empty(t, v.keys)
}
