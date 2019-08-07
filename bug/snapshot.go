package bug

import (
	"fmt"
	"time"

	"github.com/MichaelMure/git-bug/identity"
)

// Snapshot is a compiled form of the Bug data structure used for storage and merge
type Snapshot struct {
	id string

	Status       Status
	Title        string
	Comments     []Comment
	Labels       []Label
	Author       identity.Interface
	Actors       []identity.Interface
	Participants []identity.Interface
	CreatedAt    time.Time

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

// GetCreateMetadata return the creation metadata
func (snap *Snapshot) GetCreateMetadata(key string) (string, bool) {
	return snap.Operations[0].GetMetadata(key)
}

// SearchTimelineItem will search in the timeline for an item matching the given hash
func (snap *Snapshot) SearchTimelineItem(ID string) (TimelineItem, error) {
	for i := range snap.Timeline {
		if snap.Timeline[i].ID() == ID {
			return snap.Timeline[i], nil
		}
	}

	return nil, fmt.Errorf("timeline item not found")
}

// SearchComment will search for a comment matching the given hash
func (snap *Snapshot) SearchComment(id string) (*Comment, error) {
	for _, c := range snap.Comments {
		if c.id == id {
			return &c, nil
		}
	}

	return nil, fmt.Errorf("comment item not found")
}

// append the operation author to the actors list
func (snap *Snapshot) addActor(actor identity.Interface) {
	for _, a := range snap.Actors {
		if actor.Id() == a.Id() {
			return
		}
	}

	snap.Actors = append(snap.Actors, actor)
}

// append the operation author to the participants list
func (snap *Snapshot) addParticipant(participant identity.Interface) {
	for _, p := range snap.Participants {
		if participant.Id() == p.Id() {
			return
		}
	}

	snap.Participants = append(snap.Participants, participant)
}

// HasParticipant return true if the id is a participant
func (snap *Snapshot) HasParticipant(id string) bool {
	for _, p := range snap.Participants {
		if p.Id() == id {
			return true
		}
	}
	return false
}

// HasAnyParticipant return true if one of the ids is a participant
func (snap *Snapshot) HasAnyParticipant(ids ...string) bool {
	for _, id := range ids {
		if snap.HasParticipant(id) {
			return true
		}
	}
	return false
}

// HasActor return true if the id is a actor
func (snap *Snapshot) HasActor(id string) bool {
	for _, p := range snap.Actors {
		if p.Id() == id {
			return true
		}
	}
	return false
}

// HasAnyActor return true if one of the ids is a actor
func (snap *Snapshot) HasAnyActor(ids ...string) bool {
	for _, id := range ids {
		if snap.HasActor(id) {
			return true
		}
	}
	return false
}

// Sign post method for gqlgen
func (snap *Snapshot) IsAuthored() {}
