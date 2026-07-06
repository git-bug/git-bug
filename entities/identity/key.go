package identity

import (
	"encoding/json"
	"fmt"
	"strings"

	didcrypto "github.com/MetaMask/go-did-it/crypto"
	died25519 "github.com/MetaMask/go-did-it/crypto/ed25519"
	"github.com/MetaMask/go-did-it/crypto/p256"
	"github.com/MetaMask/go-did-it/crypto/p384"
	"github.com/MetaMask/go-did-it/crypto/p521"
	didrsa "github.com/MetaMask/go-did-it/crypto/rsa"
	"github.com/ProtonMail/go-crypto/openpgp"
	"golang.org/x/crypto/ssh"

	"github.com/git-bug/git-bug/repository"
)

// keySet is the allowlist of key algorithms git-bug accepts in identities.
//
// The criterion is: algorithms that git's two signing paths (SSH agent, gpg) can
// actually produce and verify signatures with.
//   - ed25519: the modern default for both SSH and GPG keys.
//   - P-256/384/521: the ecdsa-sha2-nistp* SSH key types and ECDSA GPG keys.
//   - RSA: still the majority of GPG keys and many older SSH keys, restricted to
//     the modulus sizes ssh-keygen and gpg actually generate. 2048 remains the
//     industry floor; anything smaller is weak, anything else is exotic.
//
// Notably excluded: x25519 (key exchange only, cannot sign) and secp256k1
// (no SSH or GPG counterpart).
var keySet = didcrypto.NewKeySet(
	died25519.KeyType(),
	p256.KeyType(),
	p384.KeyType(),
	p521.KeyType(),
	didrsa.KeyType(2048, 3072, 4096),
)

// KeyOrigin identifies where a Key was sourced from, determining which signer to use.
type KeyOrigin string

const (
	KeyOriginSSH KeyOrigin = "ssh"
	KeyOriginGPG KeyOrigin = "gpg"
)

// Key holds a public key in W3C DID publicKeyMultibase format.
// GPG-origin keys also store the full armored entity for verification and export.
// The private-key material is never stored — signing always delegates to the SSH
// agent or gpg subprocess.
type Key struct {
	publicKeyMultibase string
	pgpEntity          string // non-empty only for KeyOriginGPG
	origin             KeyOrigin
	signer             repository.Signer // non-nil only when set via NewKeyWithSigner (tests)
}

// NewSSHKey creates a Key from an SSH public key.
func NewSSHKey(pub ssh.PublicKey) (*Key, error) {
	multibase, err := sshPubKeyToMultibase(pub)
	if err != nil {
		return nil, err
	}
	return &Key{
		publicKeyMultibase: multibase,
		origin:             KeyOriginSSH,
	}, nil
}

// NewGPGKey creates a Key from a full armored GPG entity.
// Only the primary key's public material is extracted to produce the multibase.
func NewGPGKey(armoredEntity string) (*Key, error) {
	entities, err := openpgp.ReadArmoredKeyRing(strings.NewReader(armoredEntity))
	if err != nil {
		return nil, fmt.Errorf("parse armored GPG entity: %w", err)
	}
	if len(entities) == 0 {
		return nil, fmt.Errorf("no GPG entities found in armored input")
	}
	entity := entities[0]

	multibase, err := pgpPubKeyToMultibase(entity)
	if err != nil {
		return nil, err
	}
	return &Key{
		publicKeyMultibase: multibase,
		pgpEntity:          armoredEntity,
		origin:             KeyOriginGPG,
	}, nil
}

// NewKeyWithSigner creates a Key with an injected signer — intended for tests
// so the real SSH agent or GPG subprocess is not required.
func NewKeyWithSigner(multibase string, origin KeyOrigin, s repository.Signer) *Key {
	return &Key{
		publicKeyMultibase: multibase,
		origin:             origin,
		signer:             s,
	}
}

// PublicKeyMultibase returns the public key encoded as a W3C DID publicKeyMultibase string.
func (k *Key) PublicKeyMultibase() string {
	return k.publicKeyMultibase
}

// PGPEntity returns the stored armored GPG entity, or empty string for SSH-origin keys.
func (k *Key) PGPEntity() string {
	return k.pgpEntity
}

// Origin returns the key origin (ssh or gpg).
func (k *Key) Origin() KeyOrigin {
	return k.origin
}

// Signer returns the signer for this key.
// The injected test signer (if any) takes precedence; otherwise one is built from
// the key origin (SSH agent for SSH keys, gpg subprocess for GPG keys).
func (k *Key) Signer() (repository.Signer, error) {
	if k.signer != nil {
		return k.signer, nil
	}
	switch k.origin {
	case KeyOriginSSH:
		sshPub, err := multibaseToSSHPublicKey(k.publicKeyMultibase)
		if err != nil {
			return nil, err
		}
		return repository.NewSSHAgentSigner(sshPub), nil
	case KeyOriginGPG:
		fp, err := k.gpgFingerprint()
		if err != nil {
			return nil, err
		}
		return repository.NewGPGSigner(fp), nil
	}
	return nil, fmt.Errorf("unknown key origin: %q", k.origin)
}

// SSHPublicKey converts the stored multibase back to an ssh.PublicKey.
// Used during signature verification.
func (k *Key) SSHPublicKey() (ssh.PublicKey, error) {
	return multibaseToSSHPublicKey(k.publicKeyMultibase)
}

func (k *Key) Validate() error {
	if k.publicKeyMultibase == "" {
		return fmt.Errorf("missing public key multibase")
	}
	if k.origin == "" {
		return fmt.Errorf("missing key origin")
	}
	if k.origin == KeyOriginGPG && k.pgpEntity == "" {
		return fmt.Errorf("GPG-origin key is missing the pgp entity")
	}
	return nil
}

func (k *Key) Clone() *Key {
	return &Key{
		publicKeyMultibase: k.publicKeyMultibase,
		pgpEntity:          k.pgpEntity,
		origin:             k.origin,
		signer:             k.signer,
	}
}

type keyJSON struct {
	PublicKeyMultibase string    `json:"pub_key_multibase"`
	PGPEntity          string    `json:"pgp_entity,omitempty"`
	Origin             KeyOrigin `json:"origin"`
}

func (k *Key) MarshalJSON() ([]byte, error) {
	return json.Marshal(keyJSON{
		PublicKeyMultibase: k.publicKeyMultibase,
		PGPEntity:          k.pgpEntity,
		Origin:             k.origin,
	})
}

func (k *Key) UnmarshalJSON(data []byte) error {
	var aux keyJSON
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	k.publicKeyMultibase = aux.PublicKeyMultibase
	k.pgpEntity = aux.PGPEntity
	k.origin = aux.Origin
	return nil
}

// ---- internal helpers ----

// sshPubKeyToMultibase converts an ssh.PublicKey to a publicKeyMultibase string.
func sshPubKeyToMultibase(pub ssh.PublicKey) (string, error) {
	cp, ok := pub.(ssh.CryptoPublicKey)
	if !ok {
		return "", fmt.Errorf("unsupported SSH key type: does not implement CryptoPublicKey")
	}
	k, err := keySet.WrapPublicKey(cp.CryptoPublicKey())
	if err != nil {
		return "", err
	}
	return k.ToPublicKeyMultibase(), nil
}

// multibaseToSSHPublicKey converts a publicKeyMultibase string back to an ssh.PublicKey.
func multibaseToSSHPublicKey(mb string) (ssh.PublicKey, error) {
	pub, err := keySet.PublicKeyFromMultibase(mb)
	if err != nil {
		return nil, err
	}
	switch k := pub.(type) {
	case died25519.PublicKey:
		return ssh.NewPublicKey(k.Unwrap())
	case *p256.PublicKey:
		return ssh.NewPublicKey(k.Unwrap())
	case *p384.PublicKey:
		return ssh.NewPublicKey(k.Unwrap())
	case *p521.PublicKey:
		return ssh.NewPublicKey(k.Unwrap())
	case *didrsa.PublicKey:
		return ssh.NewPublicKey(k.Unwrap())
	default:
		return nil, fmt.Errorf("unsupported publicKeyMultibase key type: %T", k)
	}
}

// pgpPubKeyToMultibase extracts the primary public key from a GPG entity and
// converts it to a publicKeyMultibase string.
func pgpPubKeyToMultibase(entity *openpgp.Entity) (string, error) {
	k, err := keySet.WrapPublicKey(entity.PrimaryKey.PublicKey)
	if err != nil {
		return "", err
	}
	return k.ToPublicKeyMultibase(), nil
}

// gpgFingerprint parses the stored pgpEntity and returns the primary key fingerprint
// as a hex string, suitable for passing to gpg -u.
func (k *Key) gpgFingerprint() (string, error) {
	if k.pgpEntity == "" {
		return "", fmt.Errorf("no GPG entity stored for this key")
	}
	entities, err := openpgp.ReadArmoredKeyRing(strings.NewReader(k.pgpEntity))
	if err != nil {
		return "", fmt.Errorf("parse stored GPG entity: %w", err)
	}
	if len(entities) == 0 {
		return "", fmt.Errorf("no GPG entities found in stored pgpEntity")
	}
	return fmt.Sprintf("%X", entities[0].PrimaryKey.Fingerprint), nil
}
