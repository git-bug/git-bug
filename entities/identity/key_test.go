package identity

import (
	"crypto/rsa"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/repository"
)

func TestPublicKeyJSON(t *testing.T) {
	k := generatePublicKey()

	dataJSON, err := json.Marshal(k)
	require.NoError(t, err)

	var read Key
	err = json.Unmarshal(dataJSON, &read)
	require.NoError(t, err)

	require.Equal(t, k, &read)
}

func TestStoreLoad(t *testing.T) {
	repo := repository.NewMockRepoKeyring()

	// public + private
	k := GenerateKey()

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

	require.IsType(t, (*rsa.PrivateKey)(nil), k.private.PrivateKey)

	// See https://github.com/golang/crypto/pull/175
	rsaPriv := read.private.PrivateKey.(*rsa.PrivateKey)
	back := rsaPriv.Primes[0]
	rsaPriv.Primes[0] = rsaPriv.Primes[1]
	rsaPriv.Primes[1] = back

	require.True(t, k.private.PrivateKey.(*rsa.PrivateKey).Equal(read.private.PrivateKey))
}
