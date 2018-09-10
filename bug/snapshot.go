package bug

import (
	"fmt"
	"time"
)

// Snapshot is a compiled form of the Bug data structure used for storage and merge
type Snapshot struct {
	id string

	Status    Status
	Title     string
	Comments  []Comment
	Labels    []Label
	Author    Person
	CreatedAt time.Time

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
	return fmt.Sprintf("C:%d L:%d",
		len(snap.Comments)-1,
		len(snap.Labels),
	)
}

// Return the last time a bug was modified
func (snap Snapshot) LastEditTime() time.Time {
	if len(snap.Operations) == 0 {
		return time.Unix(0, 0)
	}

	return snap.Operations[len(snap.Operations)-1].Time()
}

// Return the last timestamp a bug was modified
func (snap Snapshot) LastEditUnix() int64 {
	if len(snap.Operations) == 0 {
		return 0
	}

	return snap.Operations[len(snap.Operations)-1].GetUnixTime()
}
