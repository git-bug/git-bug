package bug

import (
	"fmt"
	"time"
)

// Snapshot is a compiled form of the Bug data structure used for storage and merge
type Snapshot struct {
	id string

	Status   Status
	Title    string
	Comments []Comment
	Labels   []Label

	Operations []Operation
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
	return fmt.Sprintf("C:%d L:%d %s",
		len(snap.Comments)-1,
		len(snap.Labels),
		snap.LastEdit().Format(time.RFC822),
	)
}

// Return the last time a bug was modified
func (snap Snapshot) LastEdit() time.Time {
	if len(snap.Operations) == 0 {
		return time.Unix(0, 0)
	}

	return snap.Operations[len(snap.Operations)-1].Time()
}
