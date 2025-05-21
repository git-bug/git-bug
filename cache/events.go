package cache

import "github.com/git-bug/git-bug/entity"

type BuildEventType int

const (
	_ BuildEventType = iota
	// BuildEventCacheIsBuilt signal that the cache is being built (aka, not skipped)
	BuildEventCacheIsBuilt
	// BuildEventRemoveLock signal that an old repo lock has been cleaned
	BuildEventRemoveLock
	// BuildEventStarted signal the beginning of a cache build for an entity
	BuildEventStarted
	// BuildEventProgress signal progress in the cache building for an entity
	BuildEventProgress
	// BuildEventFinished signal the end of a cache build for an entity
	BuildEventFinished
)

// BuildEvent carry an event happening during the cache build process.
type BuildEvent struct {
	// Err carry an error if the build process failed. If set, no other field matters.
	Err error
	// Typename is the name of the entity of which the event relate to. Can be empty if no particular entity is involved.
	Typename string
	// Event is the type of the event.
	Event BuildEventType
	// Total is the total number of elements being built. Set if Event is BuildEventStarted.
	Total int64
	// Progress is the current count of processed elements. Set if Event is BuildEventProgress.
	Progress int64
}

type EntityEventType int

const (
	_ EntityEventType = iota
	EntityEventCreated
	EntityEventUpdated
	EntityEventRemoved
)

// Observer gets notified of changes in entities in the cache
type Observer interface {
	EntityEvent(event EntityEventType, repoRef string, typename string, id entity.Id)
}
