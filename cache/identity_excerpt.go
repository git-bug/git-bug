package cache

import (
	"encoding/gob"
	"fmt"
	"strings"

	"github.com/MichaelMure/git-bug/entities/identity"
	"github.com/MichaelMure/git-bug/entity"
)

// Package initialisation used to register the type for (de)serialization
func init() {
	gob.Register(IdentityExcerpt{})
}

// IdentityExcerpt hold a subset of the identity values to be able to sort and
// filter identities efficiently without having to read and compile each raw
// identity.
type IdentityExcerpt struct {
	id entity.Id

	Name              string
	Login             string
	ImmutableMetadata map[string]string
}

func NewIdentityExcerpt(i *identity.Identity) *IdentityExcerpt {
	return &IdentityExcerpt{
		id:                i.Id(),
		Name:              i.Name(),
		Login:             i.Login(),
		ImmutableMetadata: i.ImmutableMetadata(),
	}
}

func (i *IdentityExcerpt) Id() entity.Id {
	return i.id
}

// DisplayName return a non-empty string to display, representing the
// identity, based on the non-empty values.
func (i *IdentityExcerpt) DisplayName() string {
	switch {
	case i.Name == "" && i.Login != "":
		return i.Login
	case i.Name != "" && i.Login == "":
		return i.Name
	case i.Name != "" && i.Login != "":
		return fmt.Sprintf("%s (%s)", i.Name, i.Login)
	}

	panic("invalid person data")
}

// Match matches a query with the identity name, login and ID prefixes
func (i *IdentityExcerpt) Match(query string) bool {
	return i.id.HasPrefix(query) ||
		strings.Contains(strings.ToLower(i.Name), query) ||
		strings.Contains(strings.ToLower(i.Login), query)
}

/*
 * Sorting
 */

type IdentityById []*IdentityExcerpt

func (b IdentityById) Len() int {
	return len(b)
}

func (b IdentityById) Less(i, j int) bool {
	return b[i].id < b[j].id
}

func (b IdentityById) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}
