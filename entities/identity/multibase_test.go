package identity

import (
	"crypto/ed25519"
	"crypto/rand"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPubkeyMultibaseRoundTrip(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	keyBytes := []byte(priv.Public().(ed25519.PublicKey))

	encoded := pubkeyMultibaseEncode(multibaseCodeEd25519, keyBytes)
	require.True(t, strings.HasPrefix(encoded, "z"), "expected z (base58btc) prefix")

	code, decoded, err := pubkeyMultibaseDecode(encoded)
	require.NoError(t, err)
	require.Equal(t, multibaseCodeEd25519, code)
	require.Equal(t, keyBytes, decoded)
}

func TestPubkeyMultibaseRejectsNonBase58(t *testing.T) {
	// 'u' prefix = base64url, not base58btc
	_, _, err := pubkeyMultibaseDecode("uSomething")
	require.Error(t, err)
}
