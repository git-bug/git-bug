package identity

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	cryptorsa "crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ProtonMail/go-crypto/openpgp"
	"golang.org/x/crypto/ssh"

	"github.com/git-bug/git-bug/repository"
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
	switch k := cp.CryptoPublicKey().(type) {
	case ed25519.PublicKey:
		return pubkeyMultibaseEncode(multibaseCodeEd25519, k), nil
	case *ecdsa.PublicKey:
		return ecdsaPubKeyToMultibase(k)
	case *cryptorsa.PublicKey:
		der, err := x509.MarshalPKIXPublicKey(k)
		if err != nil {
			return "", err
		}
		return pubkeyMultibaseEncode(multibaseCodeRSA, der), nil
	default:
		return "", fmt.Errorf("unsupported SSH key algorithm: %T", k)
	}
}

// multibaseToSSHPublicKey converts a publicKeyMultibase string back to an ssh.PublicKey.
func multibaseToSSHPublicKey(mb string) (ssh.PublicKey, error) {
	code, keyBytes, err := pubkeyMultibaseDecode(mb)
	if err != nil {
		return nil, err
	}
	switch code {
	case multibaseCodeEd25519:
		return ssh.NewPublicKey(ed25519.PublicKey(keyBytes))
	case multibaseCodeP256:
		x, y := elliptic.UnmarshalCompressed(elliptic.P256(), keyBytes)
		if x == nil {
			return nil, fmt.Errorf("invalid P-256 public key bytes")
		}
		return ssh.NewPublicKey(&ecdsa.PublicKey{Curve: elliptic.P256(), X: x, Y: y})
	case multibaseCodeP384:
		x, y := elliptic.UnmarshalCompressed(elliptic.P384(), keyBytes)
		if x == nil {
			return nil, fmt.Errorf("invalid P-384 public key bytes")
		}
		return ssh.NewPublicKey(&ecdsa.PublicKey{Curve: elliptic.P384(), X: x, Y: y})
	case multibaseCodeP521:
		x, y := elliptic.UnmarshalCompressed(elliptic.P521(), keyBytes)
		if x == nil {
			return nil, fmt.Errorf("invalid P-521 public key bytes")
		}
		return ssh.NewPublicKey(&ecdsa.PublicKey{Curve: elliptic.P521(), X: x, Y: y})
	case multibaseCodeRSA:
		key, err := x509.ParsePKIXPublicKey(keyBytes)
		if err != nil {
			return nil, err
		}
		return ssh.NewPublicKey(key)
	default:
		return nil, fmt.Errorf("unsupported publicKeyMultibase algorithm: 0x%x", code)
	}
}

// pgpPubKeyToMultibase extracts the primary public key from a GPG entity and
// converts it to a publicKeyMultibase string.
func pgpPubKeyToMultibase(entity *openpgp.Entity) (string, error) {
	switch k := entity.PrimaryKey.PublicKey.(type) {
	case ed25519.PublicKey:
		return pubkeyMultibaseEncode(multibaseCodeEd25519, k), nil
	case *ecdsa.PublicKey:
		return ecdsaPubKeyToMultibase(k)
	case *cryptorsa.PublicKey:
		der, err := x509.MarshalPKIXPublicKey(k)
		if err != nil {
			return "", err
		}
		return pubkeyMultibaseEncode(multibaseCodeRSA, der), nil
	default:
		return "", fmt.Errorf("unsupported GPG key algorithm: %T", k)
	}
}

// ecdsaPubKeyToMultibase converts an *ecdsa.PublicKey to a publicKeyMultibase string.
func ecdsaPubKeyToMultibase(k *ecdsa.PublicKey) (string, error) {
	var code uint64
	switch k.Curve {
	case elliptic.P256():
		code = multibaseCodeP256
	case elliptic.P384():
		code = multibaseCodeP384
	case elliptic.P521():
		code = multibaseCodeP521
	default:
		return "", fmt.Errorf("unsupported ECDSA curve: %s", k.Curve.Params().Name)
	}
	return pubkeyMultibaseEncode(code, elliptic.MarshalCompressed(k.Curve, k.X, k.Y)), nil
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
