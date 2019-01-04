package bug

import (
	"fmt"
	"time"

	"github.com/MichaelMure/git-bug/util/git"
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

	Timeline []TimelineItem

	Operations []Operation
}

// Return the Bug identifier
func (snap *Snapshot) Id() string {
	return snap.id
}

// Return the Bug identifier truncated for human consumption
func (snap *Snapshot) HumanId() string {
	return FormatHumanID(snap.id)
}

// Deprecated:should be moved in UI code
func (snap *Snapshot) Summary() string {
	return fmt.Sprintf("C:%d L:%d",
		len(snap.Comments)-1,
		len(snap.Labels),
	)
}

// Return the last time a bug was modified
func (snap *Snapshot) LastEditTime() time.Time {
	if len(snap.Operations) == 0 {
		return time.Unix(0, 0)
	}

	return snap.Operations[len(snap.Operations)-1].Time()
}

// Return the last timestamp a bug was modified
func (snap *Snapshot) LastEditUnix() int64 {
	if len(snap.Operations) == 0 {
		return 0
	}

	return snap.Operations[len(snap.Operations)-1].GetUnixTime()
}

// SearchTimelineItem will search in the timeline for an item matching the given hash
func (snap *Snapshot) SearchTimelineItem(hash git.Hash) (TimelineItem, error) {
	for i := range snap.Timeline {
		if snap.Timeline[i].Hash() == hash {
			return snap.Timeline[i], nil
		}
	}

	return nil, fmt.Errorf("timeline item not found")
}
