package bug

import (
	"github.com/dustin/go-humanize"
	"time"
)

type Comment struct {
	Author  Person `json:"author"`
	Message string `json:"message"`

	// Creation time of the comment.
	// Should be used only for human display, never for ordering as we can't rely on it in a distributed system.
	UnixTime int64
}

func (c Comment) FormatTime() string {
	t := time.Unix(c.UnixTime, 0)
	return humanize.Time(t)
}
