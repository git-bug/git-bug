package identity

import (
	"encoding/json"

	"github.com/MichaelMure/git-bug/repository"
	"github.com/MichaelMure/git-bug/util/lamport"
)

var _ Interface = &IdentityStub{}

// IdentityStub is an almost empty Identity, holding only the id.
// When a normal Identity is serialized into JSON, only the id is serialized.
// All the other data are stored in git in a chain of commit + a ref.
// When this JSON is deserialized, an IdentityStub is returned instead, to be replaced
// later by the proper Identity, loaded from the Repo.
type IdentityStub struct {
	id string
}

func (i *IdentityStub) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Id string `json:"id"`
	}{
		Id: i.id,
	})
}

func (i *IdentityStub) UnmarshalJSON(data []byte) error {
	aux := struct {
		Id string `json:"id"`
	}{}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	i.id = aux.Id

	return nil
}

func (i *IdentityStub) Id() string {
	return i.id
}

func (IdentityStub) Name() string {
	panic("identities needs to be properly loaded with identity.Read()")
}

func (IdentityStub) Email() string {
	panic("identities needs to be properly loaded with identity.Read()")
}

func (IdentityStub) Login() string {
	panic("identities needs to be properly loaded with identity.Read()")
}

func (IdentityStub) AvatarUrl() string {
	panic("identities needs to be properly loaded with identity.Read()")
}

func (IdentityStub) Keys() []Key {
	panic("identities needs to be properly loaded with identity.Read()")
}

func (IdentityStub) ValidKeysAtTime(time lamport.Time) []Key {
	panic("identities needs to be properly loaded with identity.Read()")
}

func (IdentityStub) DisplayName() string {
	panic("identities needs to be properly loaded with identity.Read()")
}

func (IdentityStub) Validate() error {
	panic("identities needs to be properly loaded with identity.Read()")
}

func (IdentityStub) Commit(repo repository.Repo) error {
	panic("identities needs to be properly loaded with identity.Read()")
}

func (IdentityStub) IsProtected() bool {
	panic("identities needs to be properly loaded with identity.Read()")
}
