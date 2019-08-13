package bug

import (
	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/identity"
	"github.com/MichaelMure/git-bug/util/git"
	"github.com/MichaelMure/git-bug/util/timestamp"
	"github.com/dustin/go-humanize"
)

// Comment represent a comment in a Bug
type Comment struct {
	id      entity.Id
	Author  identity.Interface
	Message string
	Files   []git.Hash

	// Creation time of the comment.
	// Should be used only for human display, never for ordering as we can't rely on it in a distributed system.
	UnixTime timestamp.Timestamp
}

// Id return the Comment identifier
func (c Comment) Id() entity.Id {
	if c.id == "" {
		// simply panic as it would be a coding error
		// (using an id of an identity not stored yet)
		panic("no id yet")
	}
	return c.id
}

// FormatTimeRel format the UnixTime of the comment for human consumption
func (c Comment) FormatTimeRel() string {
	return humanize.Time(c.UnixTime.Time())
}

func (c Comment) FormatTime() string {
	return c.UnixTime.Time().Format("Mon Jan 2 15:04:05 2006 +0200")
}

// Sign post method for gqlgen
func (c Comment) IsAuthored() {}
