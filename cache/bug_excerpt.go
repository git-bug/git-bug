package cache

import (
	"encoding/gob"

	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/util/lamport"
)

// BugExcerpt hold a subset of the bug values to be able to sort and filter bugs
// efficiently without having to read and compile each raw bugs.
type BugExcerpt struct {
	Id string

	CreateLamportTime lamport.Time
	EditLamportTime   lamport.Time
	CreateUnixTime    int64
	EditUnixTime      int64

	Status   bug.Status
	AuthorId string
	Labels   []bug.Label

	CreateMetadata map[string]string
}

func NewBugExcerpt(b bug.Interface, snap *bug.Snapshot) *BugExcerpt {
	return &BugExcerpt{
		Id:                b.Id(),
		CreateLamportTime: b.CreateLamportTime(),
		EditLamportTime:   b.EditLamportTime(),
		CreateUnixTime:    b.FirstOp().GetUnixTime(),
		EditUnixTime:      snap.LastEditUnix(),
		Status:            snap.Status,
		AuthorId:          snap.Author.Id(),
		Labels:            snap.Labels,
		CreateMetadata:    b.FirstOp().AllMetadata(),
	}
}

// Package initialisation used to register the type for (de)serialization
func init() {
	gob.Register(BugExcerpt{})
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
