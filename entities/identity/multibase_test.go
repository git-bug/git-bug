package identity

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"strings"
	"testing"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/armor"
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

func TestKeyFromLegacyPGPJSON(t *testing.T) {
	// Generate a PGP entity — this is how git-bug's old GenerateKey would have
	// created it (openpgp.NewEntity produces RSA by default).
	entity, err := openpgp.NewEntity("test", "", "test@example.com", nil)
	require.NoError(t, err)

	// Serialize the public key in the old v2 format: armored public key block.
	var buf bytes.Buffer
	w, err := armor.Encode(&buf, openpgp.PublicKeyType, nil)
	require.NoError(t, err)
	require.NoError(t, entity.Serialize(w))
	require.NoError(t, w.Close())

	// Wrap as a JSON string, as stored in v2 version blobs.
	jsonStr, err := json.Marshal(buf.String())
	require.NoError(t, err)

	k, err := keyFromLegacyPGPJSON(json.RawMessage(jsonStr))
	require.NoError(t, err)
	require.NotEmpty(t, k.publicKeyMultibase)
	require.Equal(t, KeyOriginGPG, k.origin)
	require.Equal(t, buf.String(), k.pgpEntity, "pgpEntity should hold the armored public key so Validate() passes")
}

func TestVersionJSONLegacyV2Keys(t *testing.T) {
	// Build a v2-format version blob with an armored PGP key string.
	entity, err := openpgp.NewEntity("test", "", "test@example.com", nil)
	require.NoError(t, err)

	var buf bytes.Buffer
	w, err := armor.Encode(&buf, openpgp.PublicKeyType, nil)
	require.NoError(t, err)
	require.NoError(t, entity.Serialize(w))
	require.NoError(t, w.Close())

	v2JSON := []byte(`{
		"version": 2,
		"times": {"bugs-create": 1},
		"unix_time": 1609459200,
		"name": "Alice",
		"email": "alice@example.com",
		"pub_keys": [` + string(mustMarshalJSON(t, buf.String())) + `],
		"nonce": "AAAAAAAAAAAAAAAAAAAAAAAAAAAA"
	}`)

	var v version
	require.NoError(t, json.Unmarshal(v2JSON, &v))
	require.Len(t, v.keys, 1)
	require.NotEmpty(t, v.keys[0].publicKeyMultibase)
	require.Equal(t, KeyOriginGPG, v.keys[0].origin)
}

func mustMarshalJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return b
}
