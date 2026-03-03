package identity

import (
	"crypto/rsa"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/repository"
)



func TestPublicKeyJSON(t *testing.T) {
	id := &mockIdentity{name: "John Smith", email: "jsmith@example.com"}
	k := generatePublicKey(id)

	dataJSON, err := json.Marshal(k)
	require.NoError(t, err)

	var read Key
	err = json.Unmarshal(dataJSON, &read)
	require.NoError(t, err)

	// Compare public keys since entities may differ in internal structure after deserialization
	require.Equal(t, k.public.Fingerprint[:], read.public.Fingerprint[:])
}

func TestStoreLoad(t *testing.T) {
	repo := repository.NewMockRepoKeyring()

	// public + private
	id := &mockIdentity{name: "John Smith", email: "jsmith@example.com"}
	k := GenerateKey(id, WithTime(time.Time{}))

	// Store

	dataJSON, err := json.Marshal(k)
	require.NoError(t, err)

	err = k.storePrivate(repo)
	require.NoError(t, err)

	// Load

	var read Key
	err = json.Unmarshal(dataJSON, &read)
	require.NoError(t, err)

	err = read.ensurePrivateKey(repo)
	require.NoError(t, err)

	require.Equal(t, k.public, read.public)

	require.IsType(t, (*rsa.PrivateKey)(nil), k.entity.PrivateKey.PrivateKey)

	// See https://github.com/golang/crypto/pull/175
	rsaPriv := read.entity.PrivateKey.PrivateKey.(*rsa.PrivateKey)
	rsaPriv.Primes[0], rsaPriv.Primes[1] = rsaPriv.Primes[1], rsaPriv.Primes[0]

	require.True(t, k.entity.PrivateKey.PrivateKey.(*rsa.PrivateKey).Equal(read.entity.PrivateKey.PrivateKey))
}
