package cache

import (
	"encoding/gob"
	"fmt"
	"time"

	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/identity"
	"github.com/MichaelMure/git-bug/util/lamport"
)

// Package initialisation used to register the type for (de)serialization
func init() {
	gob.Register(BugExcerpt{})
}

// BugExcerpt hold a subset of the bug values to be able to sort and filter bugs
// efficiently without having to read and compile each raw bugs.
type BugExcerpt struct {
	Id entity.Id

	CreateLamportTime lamport.Time
	EditLamportTime   lamport.Time
	CreateUnixTime    int64
	EditUnixTime      int64

	Status       bug.Status
	Labels       []bug.Label
	Title        string
	LenComments  int
	Actors       []entity.Id
	Participants []entity.Id

	// If author is identity.Bare, LegacyAuthor is set
	// If author is identity.Identity, AuthorId is set and data is deported
	// in a IdentityExcerpt
	LegacyAuthor LegacyAuthorExcerpt
	AuthorId     entity.Id

	CreateMetadata map[string]string
}

// identity.Bare data are directly embedded in the bug excerpt
type LegacyAuthorExcerpt struct {
	Name  string
	Login string
}

func (l LegacyAuthorExcerpt) DisplayName() string {
	switch {
	case l.Name == "" && l.Login != "":
		return l.Login
	case l.Name != "" && l.Login == "":
		return l.Name
	case l.Name != "" && l.Login != "":
		return fmt.Sprintf("%s (%s)", l.Name, l.Login)
	}

	panic("invalid person data")
}

func NewBugExcerpt(b bug.Interface, snap *bug.Snapshot) *BugExcerpt {
	participantsIds := make([]entity.Id, 0, len(snap.Participants))
	for _, participant := range snap.Participants {
		if _, ok := participant.(*identity.Identity); ok {
			participantsIds = append(participantsIds, participant.Id())
		}
	}

	actorsIds := make([]entity.Id, 0, len(snap.Actors))
	for _, actor := range snap.Actors {
		if _, ok := actor.(*identity.Identity); ok {
			actorsIds = append(actorsIds, actor.Id())
		}
	}

	e := &BugExcerpt{
		Id:                b.Id(),
		CreateLamportTime: b.CreateLamportTime(),
		EditLamportTime:   b.EditLamportTime(),
		CreateUnixTime:    b.FirstOp().Time().Unix(),
		EditUnixTime:      snap.EditTime().Unix(),
		Status:            snap.Status,
		Labels:            snap.Labels,
		Actors:            actorsIds,
		Participants:      participantsIds,
		Title:             snap.Title,
		LenComments:       len(snap.Comments),
		CreateMetadata:    b.FirstOp().AllMetadata(),
	}

	switch snap.Author.(type) {
	case *identity.Identity, *IdentityCache:
		e.AuthorId = snap.Author.Id()
	case *identity.Bare:
		e.LegacyAuthor = LegacyAuthorExcerpt{
			Login: snap.Author.Login(),
			Name:  snap.Author.Name(),
		}
	default:
		panic("unhandled identity type")
	}

	return e
}

func (b *BugExcerpt) CreateTime() time.Time {
	return time.Unix(b.CreateUnixTime, 0)
}

func (b *BugExcerpt) EditTime() time.Time {
	return time.Unix(b.EditUnixTime, 0)
}

/*
 * Sorting
 */

type BugsById []*BugExcerpt

func (b BugsById) Len() int {
	return len(b)
}

func (b BugsById) Less(i, j int) bool {
	return b[i].Id < b[j].Id
}

func (b BugsById) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}

type BugsByCreationTime []*BugExcerpt

func (b BugsByCreationTime) Len() int {
	return len(b)
}

func (b BugsByCreationTime) Less(i, j int) bool {
	if b[i].CreateLamportTime < b[j].CreateLamportTime {
		return true
	}

	if b[i].CreateLamportTime > b[j].CreateLamportTime {
		return false
	}

	// When the logical clocks are identical, that means we had a concurrent
	// edition. In this case we rely on the timestamp. While the timestamp might
	// be incorrect due to a badly set clock, the drift in sorting is bounded
	// by the first sorting using the logical clock. That means that if users
	// synchronize their bugs regularly, the timestamp will rarely be used, and
	// should still provide a kinda accurate sorting when needed.
	return b[i].CreateUnixTime < b[j].CreateUnixTime
}

func (b BugsByCreationTime) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}

type BugsByEditTime []*BugExcerpt

func (b BugsByEditTime) Len() int {
	return len(b)
}

func (b BugsByEditTime) Less(i, j int) bool {
	if b[i].EditLamportTime < b[j].EditLamportTime {
		return true
	}

	if b[i].EditLamportTime > b[j].EditLamportTime {
		return false
	}

	// When the logical clocks are identical, that means we had a concurrent
	// edition. In this case we rely on the timestamp. While the timestamp might
	// be incorrect due to a badly set clock, the drift in sorting is bounded
	// by the first sorting using the logical clock. That means that if users
	// synchronize their bugs regularly, the timestamp will rarely be used, and
	// should still provide a kinda accurate sorting when needed.
	return b[i].EditUnixTime < b[j].EditUnixTime
}

func (b BugsByEditTime) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}
