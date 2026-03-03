package identity

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/armor"
	"github.com/ProtonMail/go-crypto/openpgp/packet"
	"github.com/pkg/errors"

	"github.com/git-bug/git-bug/repository"
)

var errNoPrivateKey = fmt.Errorf("no private key")

type Key struct {
	public *packet.PublicKey
	entity *openpgp.Entity
}

type keyConfig struct {
	time time.Time
}

type KeyOption func(*keyConfig)

// WithTime sets a specific time for key generation.
// Useful for testing to ensure consistent, deterministic keys.
func WithTime(t time.Time) KeyOption {
	return func(cfg *keyConfig) {
		cfg.time = t
	}
}

// GenerateKey generate a key pair (public+private) with identity metadata.
// The type and configuration of the key is determined by the default value in go's OpenPGP.
func GenerateKey(id Interface, opts ...KeyOption) *Key {
	cfg := &keyConfig{
		time: time.Now(),
	}
	for _, opt := range opts {
		opt(cfg)
	}

	entity, err := openpgp.NewEntity(id.Name(), id.Login(), id.Email(), &packet.Config{
		Time: func() time.Time {
			return cfg.time
		},
	})
	if err != nil {
		panic(err)
	}

	return &Key{
		public: entity.PrimaryKey,
		entity: entity,
	}
}

// generatePublicKey generate only a public key (only useful for testing)
// See GenerateKey for the details.
func generatePublicKey(id Interface) *Key {
	k := GenerateKey(id, WithTime(time.Time{}))
	// k.entity = nil
	k.entity.PrivateKey = nil
	return k
}

func (k *Key) Public() *packet.PublicKey {
	return k.entity.PrimaryKey
}

func (k *Key) Version() string {
	return "new"
}

func (k *Key) Private() *packet.PrivateKey {
	return k.entity.PrivateKey
}

func (k *Key) Validate() error {
	if k.public == nil {
		return fmt.Errorf("nil public key")
	}
	if !k.public.CanSign() {
		return fmt.Errorf("public key can't sign")
	}

	if k.entity != nil && k.entity.PrivateKey != nil {
		if !k.entity.PrivateKey.CanSign() {
			return fmt.Errorf("private key can't sign")
		}
	}

	return nil
}

func (k *Key) Clone() *Key {
	clone := &Key{}

	pub := *k.public
	clone.public = &pub

	if k.entity != nil {
		entity := *k.entity
		clone.entity = &entity
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

	err = k.entity.Serialize(w)
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
	// De-serialize the entity using the armored format.
	var armored string
	err := json.Unmarshal(data, &armored)
	if err != nil {
		return err
	}

	entities, err := openpgp.ReadArmoredKeyRing(strings.NewReader(armored))
	if err != nil {
		return errors.Wrap(err, "failed to read armored key ring")
	}

	if len(entities) != 1 {
		return fmt.Errorf("exactly one entity should be present - got %d", len(entities))
	}

	entity := entities[0]

	// The armored format (RFC 4880) doesn't preserve key creation timestamps.
	// We don't care about the creation time, so we reset it to the zero value to ensure consistency.
	entity.PrimaryKey.CreationTime = time.Time{}

	k.public = entity.PrimaryKey
	k.entity = entity
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

	entities, err := openpgp.ReadArmoredKeyRing(bytes.NewReader(item.Data))
	if err != nil {
		return err
	}

	if len(entities) != 1 {
		return fmt.Errorf("examtly one entity should be stored - got %d", len(entities))
	}

	// The armored format (RFC 4880) doesn't preserve key creation timestamps.
	// We don't care about the creation time, so we reset it to the zero value to ensure consistency.
	entities[0].PrivateKey.CreationTime = time.Time{}
	k.entity = entities[0]
	k.public = entities[0].PrimaryKey

	return nil
}

// ensurePrivateKey attempt to load the corresponding private key if it is not loaded already.
// If no private key is found, returns errNoPrivateKey
func (k *Key) ensurePrivateKey(repo repository.RepoKeyring) error {
	if k.entity != nil && k.entity.PrivateKey != nil {
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
	err = k.entity.SerializePrivate(w, nil)
	if err != nil {
		return err
	}
	err = w.Close()
	if err != nil {
		return err
	}

	return repo.Keyring().Set(repository.Item{
		Key:  k.entity.PrimaryKey.KeyIdString(),
		Data: buf.Bytes(),
	})
}

func (k *Key) PGPEntity() *openpgp.Entity {
	return k.entity
}
