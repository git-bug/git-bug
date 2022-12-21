package entity

type StreamedEntity[EntityT Interface] struct {
	Entity EntityT
	Err    error
}
