package identity

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/armor"
	"github.com/ProtonMail/go-crypto/openpgp/packet"
	"github.com/pkg/errors"

	"github.com/MichaelMure/git-bug/repository"
)

var errNoPrivateKey = fmt.Errorf("no private key")

type Key struct {
	public  *packet.PublicKey
	private *packet.PrivateKey
}

// GenerateKey generate a key pair (public+private)
// The type and configuration of the key is determined by the default value in go's OpenPGP.
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

func (k *Key) Public() *packet.PublicKey {
	return k.public
}

func (k *Key) Private() *packet.PrivateKey {
	return k.private
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

func (k *Key) MarshalJSON() ([]byte, error) {
	// Serialize only the public key, in the armored format.
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
	// De-serialize only the public key, in the armored format.
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

	p, err := packet.Read(block.Body)
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

func (k *Key) loadPrivate(repo repository.RepoKeyring) error {
	item, err := repo.Keyring().Get(k.public.KeyIdString())
	if err == repository.ErrKeyringKeyNotFound {
		return errNoPrivateKey
	}
	if err != nil {
		return err
	}

	block, err := armor.Decode(bytes.NewReader(item.Data))
	if err == io.EOF {
		return fmt.Errorf("no armored data found")
	}
	if err != nil {
		return err
	}

	if block.Type != openpgp.PrivateKeyType {
		return fmt.Errorf("invalid key type")
	}

	p, err := packet.Read(block.Body)
	if err != nil {
		return errors.Wrap(err, "failed to read private key packet")
	}

	private, ok := p.(*packet.PrivateKey)
	if !ok {
		return errors.New("got no packet.privateKey")
	}

	// The armored format doesn't include the creation time, which makes the round-trip data not being fully equal.
	// We don't care about the creation time so we can set it to the zero value.
	private.CreationTime = time.Time{}

	k.private = private
	return nil
}

// ensurePrivateKey attempt to load the corresponding private key if it is not loaded already.
// If no private key is found, returns errNoPrivateKey
func (k *Key) ensurePrivateKey(repo repository.RepoKeyring) error {
	if k.private != nil {
		return nil
	}

	return k.loadPrivate(repo)
}

func (k *Key) storePrivate(repo repository.RepoKeyring) error {
	var buf bytes.Buffer
	w, err := armor.Encode(&buf, openpgp.PrivateKeyType, nil)
	if err != nil {
		return err
	}
	err = k.private.Serialize(w)
	if err != nil {
		return err
	}
	err = w.Close()
	if err != nil {
		return err
	}

	return repo.Keyring().Set(repository.Item{
		Key:  k.public.KeyIdString(),
		Data: buf.Bytes(),
	})
}

func (k *Key) PGPEntity() *openpgp.Entity {
	uid := packet.NewUserId("", "", "")
	return &openpgp.Entity{
		PrimaryKey: k.public,
		PrivateKey: k.private,
		Identities: map[string]*openpgp.Identity{
			uid.Id: {
				Name:   uid.Id,
				UserId: uid,
				SelfSignature: &packet.Signature{
					IsPrimaryId: func() *bool { b := true; return &b }(),
				},
			},
		},
	}
}
