package cache

import (
	"encoding/gob"

	"github.com/MichaelMure/git-bug/identity"
)

// IdentityExcerpt hold a subset of the identity values to be able to sort and
// filter identities efficiently without having to read and compile each raw
// identity.
type IdentityExcerpt struct {
	Id string

	Name              string
	Login             string
	ImmutableMetadata map[string]string
}

func NewIdentityExcerpt(i *identity.Identity) *IdentityExcerpt {
	return &IdentityExcerpt{
		Id:                i.Id(),
		Name:              i.Name(),
		Login:             i.Login(),
		ImmutableMetadata: i.ImmutableMetadata(),
	}
}

// Package initialisation used to register the type for (de)serialization
func init() {
	gob.Register(IdentityExcerpt{})
}

/*
 * Sorting
 */

type IdentityById []*IdentityExcerpt

func (b IdentityById) Len() int {
	return len(b)
}

func (b IdentityById) Less(i, j int) bool {
	return b[i].Id < b[j].Id
}

func (b IdentityById) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}
