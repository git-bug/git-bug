package dag

import (
	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/identity"
)

// Operation is a piece of data defining a change to reflect on the state of an Entity.
// What this Operation or Entity's state looks like is not of the resort of this package as it only deals with the
// data structure and storage.
type Operation interface {
	// Id return the Operation identifier
	// Some care need to be taken to define a correct Id derivation and enough entropy in the data used to avoid
	// collisions. Notably:
	// - the Id of the first Operation will be used as the Id of the Entity. Collision need to be avoided across entities of the same type
	//   (example: no collision within the "bug" namespace).
	// - collisions can also happen within the set of Operations of an Entity. Simple Operation might not have enough
	//   entropy to yield unique Ids (example: two "close" operation within the same second, same author).
	// A common way to derive an Id will be to use the DeriveId function on the serialized operation data.
	Id() entity.Id
	// Validate check if the Operation data is valid
	Validate() error
	// Author returns the author of this operation
	Author() identity.Interface
}

// TODO: remove?
type operationBase struct {
	author identity.Interface

	// Not serialized. Store the op's id in memory.
	id entity.Id
}
