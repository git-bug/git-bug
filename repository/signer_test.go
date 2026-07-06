package repository

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

func makeSSHKeyPair(t *testing.T) (ssh.PublicKey, ed25519.PrivateKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	sshPub, err := ssh.NewPublicKey(pub)
	require.NoError(t, err)
	return sshPub, priv
}

func TestVerifySSHSIG(t *testing.T) {
	sshPub, priv := makeSSHKeyPair(t)
	ag := agent.NewKeyring().(agent.ExtendedAgent)
	require.NoError(t, ag.Add(agent.AddedKey{PrivateKey: priv}))
	signer := NewSSHAgentSignerWithAgent(sshPub, ag)

	payload := []byte("some signed payload")
	sig, err := signer.Sign(payload)
	require.NoError(t, err)

	require.NoError(t, VerifySSHSIG([]ssh.PublicKey{sshPub}, payload, sig))
	require.Error(t, VerifySSHSIG([]ssh.PublicKey{sshPub}, []byte("tampered"), sig))
}

// OpenSSH's sshsig forbids the legacy SHA-1 "ssh-rsa" algorithm: RSA keys must
// sign with rsa-sha2-512, like ssh-keygen -Y sign does.
func TestSSHAgentSignerRSAUsesSHA512(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	sshPub, err := ssh.NewPublicKey(&priv.PublicKey)
	require.NoError(t, err)
	ag := agent.NewKeyring().(agent.ExtendedAgent)
	require.NoError(t, ag.Add(agent.AddedKey{PrivateKey: priv}))
	signer := NewSSHAgentSignerWithAgent(sshPub, ag)

	payload := []byte("some signed payload")
	armored, err := signer.Sign(payload)
	require.NoError(t, err)

	_, _, sig, err := decodeSSHSIG(armored)
	require.NoError(t, err)
	require.Equal(t, ssh.KeyAlgoRSASHA512, sig.Format)

	require.NoError(t, VerifySSHSIG([]ssh.PublicKey{sshPub}, payload, armored))
}

func TestSSHAgentSignerAvailable(t *testing.T) {
	sshPub, priv := makeSSHKeyPair(t)
	ag := agent.NewKeyring().(agent.ExtendedAgent)
	require.NoError(t, ag.Add(agent.AddedKey{PrivateKey: priv}))

	require.NoError(t, NewSSHAgentSignerWithAgent(sshPub, ag).Available())

	// a key that is not loaded in the agent is not available
	otherPub, _ := makeSSHKeyPair(t)
	require.Error(t, NewSSHAgentSignerWithAgent(otherPub, ag).Available())
}

// A signature made by the right key but for another namespace must not verify:
// the namespace provides domain separation between signing contexts.
func TestVerifySSHSIGRejectsWrongNamespace(t *testing.T) {
	sshPub, priv := makeSSHKeyPair(t)
	sshSigner, err := ssh.NewSignerFromKey(priv)
	require.NoError(t, err)

	payload := []byte("some signed payload")
	const otherNamespace = "file"

	rawSig, err := sshSigner.Sign(rand.Reader, sshsigSignedData(otherNamespace, payload))
	require.NoError(t, err)
	armored, err := encodeSSHSIG(sshPub, otherNamespace, rawSig)
	require.NoError(t, err)

	err = VerifySSHSIG([]ssh.PublicKey{sshPub}, payload, armored)
	require.ErrorContains(t, err, "namespace")
}
