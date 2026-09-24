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

	testIdentity := &mockIdentity{
		name:  "name",
		email: "email",
		login: "login",
	}

	keys := []*Key{
		generatePublicKey(testIdentity),
		generatePublicKey(testIdentity),
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

	// Compare versions without Key objects since entities may differ in internal structure after deserialization
	assert.Equal(t, expected.name, after.name)
	assert.Equal(t, expected.email, after.email)
	assert.Equal(t, expected.login, after.login)
	assert.Equal(t, expected.avatarURL, after.avatarURL)
	assert.Equal(t, expected.unixTime, after.unixTime)
	assert.Equal(t, expected.times, after.times)
	assert.Equal(t, expected.metadata, after.metadata)
	assert.Equal(t, expected.id, after.id)
	assert.Equal(t, expected.commitHash, after.commitHash)
	assert.Equal(t, len(expected.keys), len(after.keys))
	for i, key := range expected.keys {
		assert.Equal(t, key.public.Fingerprint[:], after.keys[i].public.Fingerprint[:])
	}
}
