package bug

import (
	"fmt"
	"time"
)

// Snapshot is a compiled form of the Bug data structure used for storage and merge
type Snapshot struct {
	Title    string
	Comments []Comment
	Labels   []Label
}

func (snap Snapshot) Summary() string {
	return fmt.Sprintf("c:%d l:%d %s",
		len(snap.Comments)-1,
		len(snap.Labels),
		snap.LastEdit().Format(time.RFC822),
	)
}

func (snap Snapshot) LastEdit() time.Time {
	lastEditTimestamp := snap.Comments[len(snap.Comments)-1].Time
	return time.Unix(lastEditTimestamp, 0)
}
