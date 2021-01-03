package identity

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/armor"
	"golang.org/x/crypto/openpgp/packet"

	"github.com/MichaelMure/git-bug/repository"
)

type Key struct {
	public  *packet.PublicKey
	private *packet.PrivateKey
}

// GenerateKey generate a keypair (public+private)
func GenerateKey() *Key {
	entity, err := openpgp.NewEntity("", "", "", &packet.Config{
		// The armored format doesn't include the creation time, which makes the round-trip data not being fully equal.
		// We don't care about the creation time so we can set it to the zero value.
		Time: func() time.Time {
			return time.Time{}
		},
	})
	if err != nil {
		panic(err)
	}
	return &Key{
		public:  entity.PrimaryKey,
		private: entity.PrivateKey,
	}
}

// generatePublicKey generate only a public key (only useful for testing)
// See GenerateKey for the details.
func generatePublicKey() *Key {
	k := GenerateKey()
	k.private = nil
	return k
}

func (k *Key) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	w, err := armor.Encode(&buf, openpgp.PublicKeyType, nil)
	if err != nil {
		return nil, err
	}
	err = k.public.Serialize(w)
	if err != nil {
		return nil, err
	}
	err = w.Close()
	if err != nil {
		return nil, err
	}
	return json.Marshal(buf.String())
}

func (k *Key) UnmarshalJSON(data []byte) error {
	var armored string
	err := json.Unmarshal(data, &armored)
	if err != nil {
		return err
	}

	block, err := armor.Decode(strings.NewReader(armored))
	if err == io.EOF {
		return fmt.Errorf("no armored data found")
	}
	if err != nil {
		return err
	}

	if block.Type != openpgp.PublicKeyType {
		return fmt.Errorf("invalid key type")
	}

	reader := packet.NewReader(block.Body)
	p, err := reader.Next()
	if err != nil {
		return errors.Wrap(err, "failed to read public key packet")
	}

	public, ok := p.(*packet.PublicKey)
	if !ok {
		return errors.New("got no packet.publicKey")
	}

	// The armored format doesn't include the creation time, which makes the round-trip data not being fully equal.
	// We don't care about the creation time so we can set it to the zero value.
	public.CreationTime = time.Time{}

	k.public = public
	return nil
}

func (k *Key) Validate() error {
	if k.public == nil {
		return fmt.Errorf("nil public key")
	}
	if !k.public.CanSign() {
		return fmt.Errorf("public key can't sign")
	}

	if k.private != nil {
		if !k.private.CanSign() {
			return fmt.Errorf("private key can't sign")
		}
	}

	return nil
}

func (k *Key) Clone() *Key {
	clone := &Key{}

	pub := *k.public
	clone.public = &pub

	if k.private != nil {
		priv := *k.private
		clone.private = &priv
	}

	return clone
}

func (k *Key) EnsurePrivateKey(repo repository.RepoKeyring) error {
	if k.private != nil {
		return nil
	}

	// item, err := repo.Keyring().Get(k.Fingerprint())
	// if err != nil {
	// 	return fmt.Errorf("no private key found for %s", k.Fingerprint())
	// }
	//

	panic("TODO")
}

func (k *Key) Fingerprint() string {
	return string(k.public.Fingerprint[:])
}

func (k *Key) PGPEntity() *openpgp.Entity {
	return &openpgp.Entity{
		PrimaryKey: k.public,
		PrivateKey: k.private,
	}
}

var _ openpgp.KeyRing = &PGPKeyring{}

// PGPKeyring implement a openpgp.KeyRing from an slice of Key
type PGPKeyring []*Key

func (pk PGPKeyring) KeysById(id uint64) []openpgp.Key {
	var result []openpgp.Key
	for _, key := range pk {
		if key.public.KeyId == id {
			result = append(result, openpgp.Key{
				PublicKey:  key.public,
				PrivateKey: key.private,
			})
		}
	}
	return result
}

func (pk PGPKeyring) KeysByIdUsage(id uint64, requiredUsage byte) []openpgp.Key {
	// the only usage we care about is the ability to sign, which all keys should already be capable of
	return pk.KeysById(id)
}

func (pk PGPKeyring) DecryptionKeys() []openpgp.Key {
	result := make([]openpgp.Key, len(pk))
	for i, key := range pk {
		result[i] = openpgp.Key{
			PublicKey:  key.public,
			PrivateKey: key.private,
		}
	}
	return result
}
