package bug

import (
	"github.com/MichaelMure/git-bug/util/git"
	"github.com/dustin/go-humanize"
)

// Comment represent a comment in a Bug
type Comment struct {
	Author  Person
	Message string
	Files   []git.Hash

	// Creation time of the comment.
	// Should be used only for human display, never for ordering as we can't rely on it in a distributed system.
	UnixTime Timestamp
}

// FormatTimeRel format the UnixTime of the comment for human consumption
func (c Comment) FormatTimeRel() string {
	return humanize.Time(c.UnixTime.Time())
}

func (c Comment) FormatTime() string {
	return c.UnixTime.Time().Format("Mon Jan 2 15:04:05 2006 +0200")
}
