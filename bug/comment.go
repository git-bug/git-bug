package bug

import (
	"github.com/MichaelMure/git-bug/util"
	"github.com/dustin/go-humanize"
	"time"
)

// Comment represent a comment in a Bug
type Comment struct {
	Author  Person
	Message string
	Files   []util.Hash

	// Creation time of the comment.
	// Should be used only for human display, never for ordering as we can't rely on it in a distributed system.
	UnixTime int64
}

// FormatTime format the UnixTime of the comment for human consumption
func (c Comment) FormatTime() string {
	t := time.Unix(c.UnixTime, 0)
	return humanize.Time(t)
}
