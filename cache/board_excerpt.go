package cache

import (
	"time"

	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/util/lamport"
)

var _ Excerpt = &BoardExcerpt{}

// BoardExcerpt hold a subset of the board values to be able to sort and filter boards
// efficiently without having to read and compile each raw boards.
type BoardExcerpt struct {
	id entity.Id

	CreateLamportTime lamport.Time
	EditLamportTime   lamport.Time
	CreateUnixTime    int64
	EditUnixTime      int64

	Title       string
	Description string
	ItemCount   int
	Actors      []entity.Id

	CreateMetadata map[string]string
}

func newBoardExcerpt(b *sharedBoard) *BoardExcerpt {
	snap := b.Compile()

	actorsIds := make([]entity.Id, 0, len(snap.Actors))
	for _, participant := range snap.Actors {
		actorsIds = append(actorsIds, participant.Id())
	}

	return &BoardExcerpt{
		id:                b.Id(),
		CreateLamportTime: b.CreateLamportTime(),
		EditLamportTime:   b.EditLamportTime(),
		CreateUnixTime:    snap.Operations[0].Time().Unix(),
		EditUnixTime:      snap.EditTime().Unix(),
		Title:             snap.Title,
		Description:       snap.Description,
		ItemCount:         snap.ItemCount(),
		Actors:            actorsIds,
		CreateMetadata:    snap.Operations[0].AllMetadata(),
	}
}

func (b *BoardExcerpt) Id() entity.Id {
	return b.id
}

func (b *BoardExcerpt) setId(id entity.Id) {
	b.id = id
}

func (b *BoardExcerpt) CreateTime() time.Time {
	return time.Unix(b.CreateUnixTime, 0)
}

func (b *BoardExcerpt) EditTime() time.Time {
	return time.Unix(b.EditUnixTime, 0)
}
