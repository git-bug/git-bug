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
// The id of an Entity is the id of its first operation. A regular Entity is stored in
// a git reference of its Namespace named after its id: refs/<Namespace>/<id>, as part
// of the storage format (see https://github.com/git-bug/spec). Code outside of this package
// finds entities by id through ListHeads, Head and IdsFromKeys, not by reading
// references.
//
// A singleton is stored in refs/<Namespace>/<SingletonKey>, and is not addressed by
// id: ReadSingleton reads it, RemoveAll removes it.
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
	// SingletonKey, if set, makes the Entity a singleton: there is at most one in the
	// Namespace, stored in the reference with that key instead of one named after its id.
	SingletonKey string
	// a function decoding a JSON message into an Operation
	OperationUnmarshaler OperationUnmarshaler
	// the expected format version number, that can be used for data migration/upgrade
	FormatVersion uint
}

// isSingleton tells whether the Entity is a singleton, see SingletonKey.
func (def Definition) isSingleton() bool {
	return def.SingletonKey != ""
}

// keyFromId returns the key of the reference holding the Entity with this id.
func (def Definition) keyFromId(id entity.Id) string {
	if def.isSingleton() {
		return def.SingletonKey
	}
	return id.String()
}

// idFromKey tells what the reference with this key holds. If ok, it holds an Entity,
// which must have this id, empty for a singleton which can hold any Entity. If not,
// the key can't hold an Entity and the reference is ignored. The key of an Entity in
// the outdated repository format gives entity.ErrOutdatedId.
func (def Definition) idFromKey(key string) (id entity.Id, ok bool, err error) {
	if def.isSingleton() {
		return "", key == def.SingletonKey, nil
	}
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
// the id the Entity must have, empty for a singleton. See idFromKey for the references
// left out, and the error of the outdated repository format.
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
	return EditClock(def.Namespace)
}

// EditClock returns the name of the Lamport clock ordering the edition of the
// Entities of a namespace, the one of Operation.EditTime.
func EditClock(namespace string) string {
	return namespace + "-edit"
}

// entityName names the Entity with this id in messages. A singleton is named after its
// type alone, as the only one.
func (def Definition) entityName(id entity.Id) string {
	if def.isSingleton() {
		return def.Typename
	}
	return fmt.Sprintf("%s %s", def.Typename, id.Human())
}
