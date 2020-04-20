package identity

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/crypto/openpgp/armor"
	"golang.org/x/crypto/openpgp/packet"
)

// Key hold a cryptographic public key
type Key struct {
	// PubKey is the armored PGP public key.
	armoredPublicKey string

	// memoized decoded public key
	publicKey *packet.PublicKey
}

type keyJSON struct {
	ArmoredPublicKey string `json:"armored_pub_key"`
}

func NewKeyFromArmored(armoredPGPKey string) (*Key, error) {
	publicKey, err := parsePublicKey(armoredPGPKey)
	if err != nil {
		return nil, err
	}

	return &Key{armoredPGPKey, publicKey}, nil
}

func (k *Key) MarshalJSON() ([]byte, error) {
	return json.Marshal(keyJSON{
		ArmoredPublicKey: k.armoredPublicKey,
	})
}

func (k *Key) UnmarshalJSON(data []byte) error {
	var aux keyJSON

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	k.armoredPublicKey = aux.ArmoredPublicKey

	return nil
}

func (k *Key) Validate() error {
	if k.publicKey != nil {
		return nil
	}

	publicKey, err := parsePublicKey(k.armoredPublicKey)
	if err != nil {
		return errors.Wrap(err, "invalid public key")
	}
	k.publicKey = publicKey

	return nil
}

func (k *Key) Clone() *Key {
	clone := *k
	return &clone
}

func (k *Key) PublicKey() *packet.PublicKey {
	if k.publicKey != nil {
		return k.publicKey
	}

	publicKey, err := parsePublicKey(k.armoredPublicKey)
	if err != nil {
		// Coding problem, a key should be validated before use
		panic("invalid key: " + err.Error())
	}

	k.publicKey = publicKey
	return k.publicKey
}

func (k Key) Armored() string {
	return k.armoredPublicKey
}

func (k Key) Fingerprint() string {
	return encodeKeyFingerprint(k.PublicKey().Fingerprint)
}

func parsePublicKey(armoredPublicKey string) (*packet.PublicKey, error) {
	block, err := armor.Decode(strings.NewReader(armoredPublicKey))
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode armored public key")
	}

	reader := packet.NewReader(block.Body)
	p, err := reader.Next()
	if err != nil {
		return nil, errors.Wrap(err, "failed to read public key packet")
	}

	publicKey, ok := p.(*packet.PublicKey)
	if !ok {
		return nil, errors.New("got no packet.publicKey")
	}

	return publicKey, nil
}

// decodeKeyFingerprint decodes a 40 hex digits long fingerprint into bytes.
func decodeKeyFingerprint(keyFingerprint string) ([20]byte, error) {
	var fingerprint [20]byte
	fingerprintBytes, err := hex.DecodeString(keyFingerprint)
	if err != nil {
		return fingerprint, err
	}
	if len(fingerprintBytes) != 20 {
		return fingerprint, fmt.Errorf("expected 20 bytes not %d", len(fingerprintBytes))
	}
	copy(fingerprint[:], fingerprintBytes)
	return fingerprint, nil
}

// encodeKeyFingerprint encode a byte representation of a fingerprint into a 40 hex digits string.
func encodeKeyFingerprint(fingerprint [20]byte) string {
	return strings.ToUpper(hex.EncodeToString(fingerprint[:]))
}
