package identity

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	cryptorsa "crypto/rsa"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"

	"github.com/git-bug/git-bug/repository"
)

func TestKeyJSONRoundTrip(t *testing.T) {
	k := newTestKey(t)

	data, err := json.Marshal(k)
	require.NoError(t, err)

	var read Key
	require.NoError(t, json.Unmarshal(data, &read))

	require.Equal(t, k, &read)
}

func TestKeyValidate(t *testing.T) {
	k := newTestKey(t)
	require.NoError(t, k.Validate())

	require.Error(t, (&Key{origin: KeyOriginSSH}).Validate())
	require.Error(t, (&Key{publicKeyMultibase: "z..."}).Validate())
}

func TestKeySSHRoundTrip(t *testing.T) {
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	sshPub, err := ssh.NewPublicKey(pub)
	require.NoError(t, err)

	k, err := NewSSHKey(sshPub)
	require.NoError(t, err)

	recovered, err := k.SSHPublicKey()
	require.NoError(t, err)

	require.Equal(t, sshPub.Marshal(), recovered.Marshal())
}

func TestKeySSHECDSARoundTrip(t *testing.T) {
	for _, curve := range []elliptic.Curve{elliptic.P256(), elliptic.P384(), elliptic.P521()} {
		priv, err := ecdsa.GenerateKey(curve, rand.Reader)
		require.NoError(t, err)
		sshPub, err := ssh.NewPublicKey(&priv.PublicKey)
		require.NoError(t, err)

		k, err := NewSSHKey(sshPub)
		require.NoError(t, err)

		recovered, err := k.SSHPublicKey()
		require.NoError(t, err)

		require.Equal(t, sshPub.Marshal(), recovered.Marshal())
	}
}

func TestKeySSHRSARoundTrip(t *testing.T) {
	priv, err := cryptorsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	sshPub, err := ssh.NewPublicKey(&priv.PublicKey)
	require.NoError(t, err)

	k, err := NewSSHKey(sshPub)
	require.NoError(t, err)

	recovered, err := k.SSHPublicKey()
	require.NoError(t, err)

	require.Equal(t, sshPub.Marshal(), recovered.Marshal())
}

// newTestKey builds a public-only SSH key for use in structural tests that don't
// involve signing (e.g. JSON round-trip, ValidKeysAtTime). The signer field is nil,
// which matches what UnmarshalJSON produces, so require.Equal works correctly.
func newTestKey(t *testing.T) *Key {
	t.Helper()
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	sshPub, err := ssh.NewPublicKey(pub)
	require.NoError(t, err)
	k, err := NewSSHKey(sshPub)
	require.NoError(t, err)
	return k
}

// newTestSigningKey builds an Ed25519 key backed by an in-process SSH agent,
// for use in tests that actually call Signer().
func newTestSigningKey(t *testing.T) *Key {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	sshPub, err := ssh.NewPublicKey(pub)
	require.NoError(t, err)
	ag := agent.NewKeyring().(agent.ExtendedAgent)
	require.NoError(t, ag.Add(agent.AddedKey{PrivateKey: priv}))
	signer := repository.NewSSHAgentSignerWithAgent(sshPub, ag)
	k, err := NewSSHKey(sshPub)
	require.NoError(t, err)
	return NewKeyWithSigner(k.PublicKeyMultibase(), KeyOriginSSH, signer)
}
