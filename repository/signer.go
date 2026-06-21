package repository

import (
	"bytes"
	"crypto/sha512"
	"encoding/base64"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// Signer produces a detached signature over a payload.
// The returned bytes are an armored string (either PGP or SSHSIG), ready to be
// stored in a git commit's gpgsig header.
type Signer interface {
	Sign(payload []byte) (signature []byte, err error)

	// Available reports whether the signing key is usable right now (loaded in the
	// SSH agent, present in the gpg keyring ...). It allows the caller to fall back
	// to another key instead of failing later at signing time.
	Available() error
}

// SSHAgentSigner signs via an SSH agent. In production it dials $SSH_AUTH_SOCK.
// In tests, an in-memory agent can be injected via NewSSHAgentSignerWithAgent.
type SSHAgentSigner struct {
	pubKey        ssh.PublicKey
	agentOverride agent.ExtendedAgent // nil = dial $SSH_AUTH_SOCK
}

func NewSSHAgentSigner(pubKey ssh.PublicKey) *SSHAgentSigner {
	return &SSHAgentSigner{pubKey: pubKey}
}

// NewSSHAgentSignerWithAgent creates a signer backed by a pre-built agent — for tests.
func NewSSHAgentSignerWithAgent(pubKey ssh.PublicKey, ag agent.ExtendedAgent) *SSHAgentSigner {
	return &SSHAgentSigner{pubKey: pubKey, agentOverride: ag}
}

func (s *SSHAgentSigner) getAgent() (agent.ExtendedAgent, func(), error) {
	if s.agentOverride != nil {
		return s.agentOverride, func() {}, nil
	}
	sockPath := os.Getenv("SSH_AUTH_SOCK")
	if sockPath == "" {
		return nil, nil, fmt.Errorf("SSH_AUTH_SOCK not set; run ssh-add to load your key")
	}
	conn, err := net.Dial("unix", sockPath)
	if err != nil {
		return nil, nil, fmt.Errorf("dial SSH agent: %w", err)
	}
	return agent.NewClient(conn), func() { conn.Close() }, nil
}

func (s *SSHAgentSigner) Sign(payload []byte) ([]byte, error) {
	ag, cleanup, err := s.getAgent()
	if err != nil {
		return nil, err
	}
	defer cleanup()

	// OpenSSH's sshsig forbids the legacy SHA-1 "ssh-rsa" algorithm that agents
	// use by default for RSA keys; request rsa-sha2-512 instead, matching what
	// ssh-keygen -Y sign produces.
	var flags agent.SignatureFlags
	if s.pubKey.Type() == ssh.KeyAlgoRSA {
		flags = agent.SignatureFlagRsaSha512
	}

	signedData := sshsigSignedData(sshsigNamespace, payload)
	sig, err := ag.SignWithFlags(s.pubKey, signedData, flags)
	if err != nil {
		return nil, fmt.Errorf("SSH agent sign: %w", err)
	}
	return encodeSSHSIG(s.pubKey, sshsigNamespace, sig)
}

// Available reports whether the key is currently loaded in the SSH agent.
func (s *SSHAgentSigner) Available() error {
	ag, cleanup, err := s.getAgent()
	if err != nil {
		return err
	}
	defer cleanup()

	agentKeys, err := ag.List()
	if err != nil {
		return fmt.Errorf("list SSH agent keys: %w", err)
	}
	want := s.pubKey.Marshal()
	for _, k := range agentKeys {
		if bytes.Equal(k.Marshal(), want) {
			return nil
		}
	}
	return fmt.Errorf("key %s is not loaded in the SSH agent; run ssh-add", ssh.FingerprintSHA256(s.pubKey))
}

// GPGSigner signs via the gpg subprocess, using the key identified by fingerprint.
type GPGSigner struct {
	fingerprint string // hex fingerprint or key ID, passed to gpg -u
}

func NewGPGSigner(fingerprint string) *GPGSigner {
	return &GPGSigner{fingerprint: fingerprint}
}

func (g *GPGSigner) Sign(payload []byte) ([]byte, error) {
	var stderr bytes.Buffer
	cmd := exec.Command("gpg", "--armor", "--detach-sign", "-u", g.fingerprint)
	cmd.Stdin = bytes.NewReader(payload)
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		if stderr.Len() > 0 {
			return nil, fmt.Errorf("gpg sign: %w: %s", err, strings.TrimSpace(stderr.String()))
		}
		return nil, fmt.Errorf("gpg sign: %w", err)
	}
	return out, nil
}

// Available reports whether gpg knows a secret key for the fingerprint.
func (g *GPGSigner) Available() error {
	var stderr bytes.Buffer
	cmd := exec.Command("gpg", "--batch", "--list-secret-keys", g.fingerprint)
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return fmt.Errorf("no gpg secret key for %s: %s", g.fingerprint, strings.TrimSpace(stderr.String()))
		}
		return fmt.Errorf("no gpg secret key for %s: %w", g.fingerprint, err)
	}
	return nil
}

// VerifySSHSIG verifies an armored SSHSIG against a set of SSH public keys.
// The pubKeys slice is matched against the key embedded in the SSHSIG.
func VerifySSHSIG(pubKeys []ssh.PublicKey, signedData, armoredSSHSIG []byte) error {
	sigPubKey, namespace, sig, err := decodeSSHSIG(armoredSSHSIG)
	if err != nil {
		return fmt.Errorf("parse SSHSIG: %w", err)
	}

	// PROTOCOL.sshsig requires the verifier to check the namespace: without this,
	// a signature the key owner made for any other purpose would verify here.
	if namespace != sshsigNamespace {
		return fmt.Errorf("SSHSIG namespace %q, expected %q", namespace, sshsigNamespace)
	}

	// Verify the signature key matches one of the expected keys.
	sigPubKeyBytes := sigPubKey.Marshal()
	found := false
	for _, k := range pubKeys {
		if bytes.Equal(k.Marshal(), sigPubKeyBytes) {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("SSHSIG key does not match any valid identity key")
	}

	blob := sshsigSignedData(namespace, signedData)
	return sigPubKey.Verify(blob, sig)
}

// IsSSHSignature reports whether signature is an SSHSIG armored block.
func IsSSHSignature(signature []byte) bool {
	return bytes.HasPrefix(bytes.TrimSpace(signature), []byte("-----BEGIN SSH SIGNATURE-----"))
}

// ---- SSHSIG binary format (PROTOCOL.sshsig) ----

const sshsigMagic = "SSHSIG"
const sshsigNamespace = "git"
const sshsigHashAlgo = "sha512"
const sshsigVersion = uint32(1)

// sshsigSignedData builds the blob that the SSH key actually signs, per PROTOCOL.sshsig.
func sshsigSignedData(namespace string, payload []byte) []byte {
	h := sha512.Sum512(payload)

	type blobFields struct {
		Namespace string
		Reserved  string
		HashAlgo  string
		Hash      []byte
	}
	inner := ssh.Marshal(blobFields{
		Namespace: namespace,
		Reserved:  "",
		HashAlgo:  sshsigHashAlgo,
		Hash:      h[:],
	})
	return append([]byte(sshsigMagic), inner...)
}

// encodeSSHSIG produces the armored SSHSIG block stored in the git commit.
func encodeSSHSIG(pubKey ssh.PublicKey, namespace string, sig *ssh.Signature) ([]byte, error) {
	type innerSig struct {
		Format string
		Blob   []byte
	}
	type blobFields struct {
		Version   uint32
		PublicKey []byte
		Namespace string
		Reserved  string
		HashAlgo  string
		Signature []byte
	}
	raw := append([]byte(sshsigMagic), ssh.Marshal(blobFields{
		Version:   sshsigVersion,
		PublicKey: pubKey.Marshal(),
		Namespace: namespace,
		Reserved:  "",
		HashAlgo:  sshsigHashAlgo,
		Signature: ssh.Marshal(innerSig{Format: sig.Format, Blob: sig.Blob}),
	})...)

	encoded := base64.StdEncoding.EncodeToString(raw)
	var buf strings.Builder
	buf.WriteString("-----BEGIN SSH SIGNATURE-----\n")
	for len(encoded) > 0 {
		n := 70
		if n > len(encoded) {
			n = len(encoded)
		}
		buf.WriteString(encoded[:n])
		buf.WriteByte('\n')
		encoded = encoded[n:]
	}
	buf.WriteString("-----END SSH SIGNATURE-----\n")
	return []byte(buf.String()), nil
}

// decodeSSHSIG parses an armored SSHSIG block.
func decodeSSHSIG(armored []byte) (pubKey ssh.PublicKey, namespace string, sig *ssh.Signature, err error) {
	s := strings.TrimSpace(string(armored))
	s = strings.TrimPrefix(s, "-----BEGIN SSH SIGNATURE-----")
	s = strings.TrimSuffix(strings.TrimSpace(s), "-----END SSH SIGNATURE-----")
	s = strings.ReplaceAll(strings.TrimSpace(s), "\n", "")

	raw, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, "", nil, fmt.Errorf("base64 decode: %w", err)
	}
	if !bytes.HasPrefix(raw, []byte(sshsigMagic)) {
		return nil, "", nil, fmt.Errorf("missing SSHSIG magic")
	}
	raw = raw[len(sshsigMagic):]

	type blobFields struct {
		Version   uint32
		PublicKey []byte
		Namespace string
		Reserved  string
		HashAlgo  string
		Signature []byte
	}
	var blob blobFields
	if err := ssh.Unmarshal(raw, &blob); err != nil {
		return nil, "", nil, fmt.Errorf("unmarshal SSHSIG: %w", err)
	}
	if blob.Version != sshsigVersion {
		return nil, "", nil, fmt.Errorf("unsupported SSHSIG version %d", blob.Version)
	}
	if blob.HashAlgo != sshsigHashAlgo {
		return nil, "", nil, fmt.Errorf("unsupported SSHSIG hash algorithm %q", blob.HashAlgo)
	}

	pubKey, err = ssh.ParsePublicKey(blob.PublicKey)
	if err != nil {
		return nil, "", nil, fmt.Errorf("parse SSHSIG public key: %w", err)
	}

	type innerSig struct {
		Format string
		Blob   []byte
	}
	var iSig innerSig
	if err := ssh.Unmarshal(blob.Signature, &iSig); err != nil {
		return nil, "", nil, fmt.Errorf("parse SSHSIG signature: %w", err)
	}

	return pubKey, blob.Namespace, &ssh.Signature{Format: iSig.Format, Blob: iSig.Blob}, nil
}
