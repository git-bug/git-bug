package entity

type StreamedEntity[EntityT Interface] struct {
	Err    error
	Entity EntityT

	// CurrentEntity is the index of the current entity being streamed, to express progress.
	CurrentEntity int64
	// TotalEntities is the total count of expected entities, if known.
	TotalEntities int64
}
