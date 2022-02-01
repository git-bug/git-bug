package gitlab

import (
	"fmt"
	"strings"
	"time"

	"github.com/MichaelMure/git-bug/util/text"
	"github.com/xanzy/go-gitlab"
)

// Event represents a unified GitLab event (note, label or state event).
type Event interface {
	ID() string
	UserID() int
	Kind() EventKind
	CreatedAt() time.Time
}

type EventKind int

const (
	EventUnknown EventKind = iota
	EventError
	EventComment
	EventTitleChanged
	EventDescriptionChanged
	EventClosed
	EventReopened
	EventLocked
	EventUnlocked
	EventChangedDuedate
	EventRemovedDuedate
	EventAssigned
	EventUnassigned
	EventChangedMilestone
	EventRemovedMilestone
	EventAddLabel
	EventRemoveLabel
	EventMentionedInIssue
	EventMentionedInMergeRequest
)

var _ Event = &NoteEvent{}

type NoteEvent struct{ gitlab.Note }

func (n NoteEvent) ID() string           { return fmt.Sprintf("%d", n.Note.ID) }
func (n NoteEvent) UserID() int          { return n.Author.ID }
func (n NoteEvent) CreatedAt() time.Time { return *n.Note.CreatedAt }
func (n NoteEvent) Kind() EventKind {

	switch {
	case !n.System:
		return EventComment

	case n.Body == "closed":
		return EventClosed

	case n.Body == "reopened":
		return EventReopened

	case n.Body == "changed the description":
		return EventDescriptionChanged

	case n.Body == "locked this issue":
		return EventLocked

	case n.Body == "unlocked this issue":
		return EventUnlocked

	case strings.HasPrefix(n.Body, "changed title from"):
		return EventTitleChanged

	case strings.HasPrefix(n.Body, "changed due date to"):
		return EventChangedDuedate

	case n.Body == "removed due date":
		return EventRemovedDuedate

	case strings.HasPrefix(n.Body, "assigned to @"):
		return EventAssigned

	case strings.HasPrefix(n.Body, "unassigned @"):
		return EventUnassigned

	case strings.HasPrefix(n.Body, "changed milestone to %"):
		return EventChangedMilestone

	case strings.HasPrefix(n.Body, "removed milestone"):
		return EventRemovedMilestone

	case strings.HasPrefix(n.Body, "mentioned in issue"):
		return EventMentionedInIssue

	case strings.HasPrefix(n.Body, "mentioned in merge request"):
		return EventMentionedInMergeRequest

	default:
		return EventUnknown
	}

}

func (n NoteEvent) Title() string {
	if n.Kind() == EventTitleChanged {
		return getNewTitle(n.Body)
	}
	return text.CleanupOneLine(n.Body)
}

var _ Event = &LabelEvent{}

type LabelEvent struct{ gitlab.LabelEvent }

func (l LabelEvent) ID() string           { return fmt.Sprintf("%d", l.LabelEvent.ID) }
func (l LabelEvent) UserID() int          { return l.User.ID }
func (l LabelEvent) CreatedAt() time.Time { return *l.LabelEvent.CreatedAt }
func (l LabelEvent) Kind() EventKind {
	switch l.Action {
	case "add":
		return EventAddLabel
	case "remove":
		return EventRemoveLabel
	default:
		return EventUnknown
	}
}

var _ Event = &StateEvent{}

type StateEvent struct{ gitlab.StateEvent }

func (s StateEvent) ID() string           { return fmt.Sprintf("%d", s.StateEvent.ID) }
func (s StateEvent) UserID() int          { return s.User.ID }
func (s StateEvent) CreatedAt() time.Time { return *s.StateEvent.CreatedAt }
func (s StateEvent) Kind() EventKind {
	switch s.State {
	case "closed":
		return EventClosed
	case "opened", "reopened":
		return EventReopened
	default:
		return EventUnknown
	}
}

var _ Event = &ErrorEvent{}

type ErrorEvent struct {
	Err  error
	Time time.Time
}

func (e ErrorEvent) ID() string           { return "" }
func (e ErrorEvent) UserID() int          { return -1 }
func (e ErrorEvent) CreatedAt() time.Time { return e.Time }
func (e ErrorEvent) Kind() EventKind      { return EventError }

// SortedEvents fan-in some Event-channels into one, sorted by creation date, using CreatedAt-method.
// This function assume that each channel is pre-ordered.
func SortedEvents(inputs ...<-chan Event) chan Event {
	out := make(chan Event)

	go func() {
		defer close(out)

		heads := make([]Event, len(inputs))

		// pre-fill the head view
		for i, input := range inputs {
			if event, ok := <-input; ok {
				heads[i] = event
			}
		}

		for {
			var earliestEvent Event
			var originChannel int

			// pick the earliest event of the heads
			for i, head := range heads {
				if head != nil && (earliestEvent == nil || head.CreatedAt().Before(earliestEvent.CreatedAt())) {
					earliestEvent = head
					originChannel = i
				}
			}

			if earliestEvent == nil {
				// no event anymore, we are done
				return
			}

			// we have an event: consume it and replace it if possible
			heads[originChannel] = nil
			if event, ok := <-inputs[originChannel]; ok {
				heads[originChannel] = event
			}
			out <- earliestEvent
		}
	}()

	return out
}

// getNewTitle parses body diff given by gitlab api and return it final form
// examples: "changed title from **fourth issue** to **fourth issue{+ changed+}**"
//           "changed title from **fourth issue{- changed-}** to **fourth issue**"
// because Gitlab
func getNewTitle(diff string) string {
	newTitle := strings.Split(diff, "** to **")[1]
	newTitle = strings.Replace(newTitle, "{+", "", -1)
	newTitle = strings.Replace(newTitle, "+}", "", -1)
	return strings.TrimSuffix(newTitle, "**")
}
