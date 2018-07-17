package bug

import (
	"fmt"
	"time"
)

// Snapshot is a compiled form of the Bug data structure used for storage and merge
type Snapshot struct {
	id       string
	Status   Status
	Title    string
	Comments []Comment
	Labels   []Label
}

// Return the Bug identifier
func (snap Snapshot) Id() string {
	return snap.id
}

// Return the Bug identifier truncated for human consumption
func (snap Snapshot) HumanId() string {
	return fmt.Sprintf("%.8s", snap.id)
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
