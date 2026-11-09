package dag

import (
	"context"
	"encoding/json"

	"github.com/git-bug/git-bug/entity"
)

type OperationUnmarshaler func(ctx context.Context, raw json.RawMessage, resolver entity.Resolvers) (Operation, error)

// Definition hold the details defining one specialization of an Entity.
type Definition struct {
	// the name of the entity (bug, pull-request, ...), for human consumption
	Typename string
	// the Namespace in git references (bugs, prs, ...)
	Namespace string
	// a function decoding a JSON message into an Operation
	OperationUnmarshaler OperationUnmarshaler
	// the expected format version number, that can be used for data migration/upgrade
	FormatVersion uint
}

// creationClock returns the name of the Lamport clock ordering the creation of the
// Entities.
func (def Definition) creationClock() string {
	return def.Namespace + "-create"
}

// editClock returns the name of the Lamport clock ordering the edition of the Entities.
func (def Definition) editClock() string {
	return def.Namespace + "-edit"
}
