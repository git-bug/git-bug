package bug

import "time"

type Comment struct {
	Author  Person
	Message string

	// Creation time of the comment.
	// Should be used only for human display, never for ordering as we can't rely on it in a distributed system.
	Time int64
}

func (c Comment) FormatTime() string {
	return time.Unix(c.Time, 0).Format(time.RFC822)
}
