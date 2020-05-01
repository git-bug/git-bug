package identity

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/crypto/openpgp/armor"
	"golang.org/x/crypto/openpgp/packet"
)

type Key struct {
	// PubKey is the armored PGP public key.
	ArmoredPublicKey string `json:"pub_key"`

	PublicKey *packet.PublicKey `json:"-"`
}

func NewKey(armoredPGPKey string) (*Key, error) {
	publicKey, err := parsePublicKey(armoredPGPKey)
	if err != nil {
		return nil, err
	}

	return &Key{armoredPGPKey, publicKey}, nil
}

func parsePublicKey(armoredPublicKey string) (*packet.PublicKey, error) {
	block, err := armor.Decode(strings.NewReader(armoredPublicKey))
	if err != nil {
		return nil, errors.Wrap(err, "failed to dearmor public key")
	}

	reader := packet.NewReader(block.Body)
	p, err := reader.Next()
	if err != nil {
		return nil, errors.Wrap(err, "failed to read public key packet")
	}

	publicKey, ok := p.(*packet.PublicKey)
	if !ok {
		return nil, errors.New("got no packet.PublicKey")
	}

	return publicKey, nil
}

// DecodeKeyFingerprint decodes a 40 hex digits long fingerprint into bytes.
func DecodeKeyFingerprint(keyFingerprint string) ([20]byte, error) {
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

func EncodeKeyFingerprint(fingerprint [20]byte) string {
	return strings.ToUpper(hex.EncodeToString(fingerprint[:]))
}

func (k *Key) Validate() error {
	_, err := k.GetPublicKey()
	return err
}

func (k *Key) Clone() *Key {
	clone := *k
	return &clone
}

func (k *Key) GetPublicKey() (*packet.PublicKey, error) {
	var err error
	if k.PublicKey == nil {
		k.PublicKey, err = parsePublicKey(k.ArmoredPublicKey)
	}
	return k.PublicKey, err
}
