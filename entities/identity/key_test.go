package identity

import (
	"crypto/rsa"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/repository"
	"github.com/git-bug/git-bug/util/lamport"
	"github.com/git-bug/git-bug/util/timestamp"
)

type mockIdentity struct {
	name  string
	login string
	email string
}

func (m *mockIdentity) Name() string                                        { return m.name }
func (m *mockIdentity) Login() string                                      { return m.login }
func (m *mockIdentity) Email() string                                     { return m.email }
func (m *mockIdentity) DisplayName() string                               { return m.name }
func (m *mockIdentity) AvatarUrl() string                                 { return "" }
func (m *mockIdentity) Keys() []*Key                                      { return nil }
func (m *mockIdentity) SigningKey(repo repository.RepoKeyring) (*Key, error) { return nil, nil }
func (m *mockIdentity) ValidKeysAtTime(clockName string, time lamport.Time) []*Key { return nil }
func (m *mockIdentity) LastModification() timestamp.Timestamp             { return 0 }
func (m *mockIdentity) LastModificationLamports() map[string]lamport.Time { return nil }
func (m *mockIdentity) IsProtected() bool                                 { return false }
func (m *mockIdentity) Validate() error                                   { return nil }
func (m *mockIdentity) NeedCommit() bool                                  { return false }
func (m *mockIdentity) Id() entity.Id                                     { return "" }

func TestPublicKeyJSON(t *testing.T) {
	id := &mockIdentity{name: "John Smith", email: "jsmith@example.com"}
	k := generatePublicKey(id)

	dataJSON, err := json.Marshal(k)
	require.NoError(t, err)

	var read Key
	err = json.Unmarshal(dataJSON, &read)
	require.NoError(t, err)

	// Compare public keys since entities may differ in internal structure after deserialization
	require.Equal(t, k.public.Fingerprint[:], read.public.Fingerprint[:])
}

func TestStoreLoad(t *testing.T) {
	repo := repository.NewMockRepoKeyring()

	// public + private
	id := &mockIdentity{name: "John Smith", email: "jsmith@example.com"}
	k := GenerateKey(id, WithTime(time.Time{}))

	// Store

	dataJSON, err := json.Marshal(k)
	require.NoError(t, err)

	err = k.storePrivate(repo)
	require.NoError(t, err)

	// Load

	var read Key
	err = json.Unmarshal(dataJSON, &read)
	require.NoError(t, err)

	err = read.ensurePrivateKey(repo)
	require.NoError(t, err)

	require.Equal(t, k.public, read.public)

	require.IsType(t, (*rsa.PrivateKey)(nil), k.entity.PrivateKey.PrivateKey)

	// See https://github.com/golang/crypto/pull/175
	rsaPriv := read.entity.PrivateKey.PrivateKey.(*rsa.PrivateKey)
	rsaPriv.Primes[0], rsaPriv.Primes[1] = rsaPriv.Primes[1], rsaPriv.Primes[0]

	require.True(t, k.entity.PrivateKey.PrivateKey.(*rsa.PrivateKey).Equal(read.entity.PrivateKey.PrivateKey))
}
