package cache

import (
	"encoding/gob"
	"strings"

	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/identity"
)

// Package initialisation used to register the type for (de)serialization
func init() {
	gob.Register(IdentityExcerpt{})
}

// IdentityExcerpt hold a subset of the identity values to be able to sort and
// filter identities efficiently without having to read and compile each raw
// identity.
type IdentityExcerpt struct {
	Id entity.Id

	Name              string
	ImmutableMetadata map[string]string
}

func NewIdentityExcerpt(i *identity.Identity) *IdentityExcerpt {
	return &IdentityExcerpt{
		Id:                i.Id(),
		Name:              i.Name(),
		ImmutableMetadata: i.ImmutableMetadata(),
	}
}

// DisplayName return a non-empty string to display, representing the
// identity, based on the non-empty values.
func (i *IdentityExcerpt) DisplayName() string {
	return i.Name
}

// Match matches a query with the identity name, login and ID prefixes
func (i *IdentityExcerpt) Match(query string) bool {
	return i.Id.HasPrefix(query) ||
		strings.Contains(strings.ToLower(i.Name), query)
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
