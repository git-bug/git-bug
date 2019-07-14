package gitlab

import (
	"strings"

	"github.com/xanzy/go-gitlab"
)

type NoteType int

const (
	_ NoteType = iota
	NOTE_COMMENT
	NOTE_TITLE_CHANGED
	NOTE_DESCRIPTION_CHANGED
	NOTE_CLOSED
	NOTE_REOPENED
	NOTE_LOCKED
	NOTE_UNLOCKED
	NOTE_CHANGED_DUEDATE
	NOTE_REMOVED_DUEDATE
	NOTE_ASSIGNED
	NOTE_UNASSIGNED
	NOTE_CHANGED_MILESTONE
	NOTE_REMOVED_MILESTONE
	NOTE_UNKNOWN
)

// GetNoteType parses note body a give it type
// Since gitlab api return all these NoteType event as the same object
// and doesn't provide a field to specify the note type. We must parse the
// note body to detect it type.
func GetNoteType(n *gitlab.Note) (NoteType, string) {
	if n.Body == "closed" {
		return NOTE_CLOSED, ""
	}

	if n.Body == "reopened" {
		return NOTE_REOPENED, ""
	}

	if n.Body == "changed the description" {
		return NOTE_DESCRIPTION_CHANGED, ""
	}

	if n.Body == "locked this issue" {
		return NOTE_LOCKED, ""
	}

	if n.Body == "unlocked this issue" {
		return NOTE_UNLOCKED, ""
	}

	if strings.HasPrefix(n.Body, "changed title from") {
		return NOTE_TITLE_CHANGED, getNewTitle(n.Body)
	}

	if strings.HasPrefix(n.Body, "changed due date to") {
		return NOTE_CHANGED_DUEDATE, ""
	}

	if n.Body == "removed due date" {
		return NOTE_REMOVED_DUEDATE, ""
	}

	if strings.HasPrefix(n.Body, "assigned to @") {
		return NOTE_ASSIGNED, ""
	}

	if strings.HasPrefix(n.Body, "unassigned @") {
		return NOTE_UNASSIGNED, ""
	}

	if strings.HasPrefix(n.Body, "changed milestone to %") {
		return NOTE_CHANGED_MILESTONE, ""
	}

	if strings.HasPrefix(n.Body, "removed milestone") {
		return NOTE_REMOVED_MILESTONE, ""
	}

	// comment don't have a specific format
	return NOTE_COMMENT, n.Body
}

// getNewTitle parses body diff given by gitlab api and return it final form
// examples: "changed title from **fourth issue** to **fourth issue{+ changed+}**"
//           "changed title from **fourth issue{- changed-}** to **fourth issue**"
func getNewTitle(diff string) string {
	newTitle := strings.Split(diff, "** to **")[1]
	newTitle = strings.Replace(newTitle, "{+", "", -1)
	newTitle = strings.Replace(newTitle, "+}", "", -1)
	return strings.TrimSuffix(newTitle, "**")
}
