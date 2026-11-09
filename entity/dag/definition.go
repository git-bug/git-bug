package dag

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/repository"
)

type OperationUnmarshaler func(ctx context.Context, raw json.RawMessage, resolver entity.Resolvers) (Operation, error)

// Definition hold the details defining one specialization of an Entity.
//
// The id of an Entity is the id of its first operation. An Entity is stored in a git
// reference of its Namespace named after its id: refs/<Namespace>/<id>, as part of the
// storage format (see https://github.com/git-bug/spec). Code outside of this package finds
// entities by id through ListHeads, Head and IdsFromKeys, not by reading references.
//
// A reference of the Namespace whose key can't hold an Entity is not an Entity: every
// function ignores it, locally or on a remote. The exception is the key of an Entity
// in the outdated repository format, which gives entity.ErrOutdatedId, for the
// repository to be upgraded.
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

// idFromKey tells what the reference with this key holds. If ok, it holds the Entity
// with this id. If not, the key can't hold an Entity and the reference is ignored. The
// key of an Entity in the outdated repository format gives entity.ErrOutdatedId.
func (def Definition) idFromKey(key string) (id entity.Id, ok bool, err error) {
	id = entity.Id(key)
	err = id.Validate()
	switch {
	case errors.Is(err, entity.ErrOutdatedId):
		return "", false, err
	case err != nil:
		return "", false, nil
	}
	return id, true, nil
}

// listEntities returns the commit the reference of each local Entity points to, by
// its id. See idFromKey for the references left out, and the error of the outdated
// repository format.
func (def Definition) listEntities(ctx context.Context, repo repository.RepoData) (map[entity.Id]repository.Hash, error) {
	refs, err := repo.ListRefs(ctx, def.Namespace)
	if err != nil {
		return nil, err
	}
	heads := make(map[entity.Id]repository.Hash, len(refs))
	for key, commit := range refs {
		id, ok, err := def.idFromKey(key)
		if err != nil {
			return nil, fmt.Errorf("invalid id %q: %w", key, err)
		}
		if ok {
			heads[id] = commit
		}
	}
	return heads, nil
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

// entityName names the Entity with this id in messages.
func (def Definition) entityName(id entity.Id) string {
	return fmt.Sprintf("%s %s", def.Typename, id.Human())
}
