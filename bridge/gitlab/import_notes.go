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
	NOTE_MENTIONED_IN_ISSUE
	NOTE_MENTIONED_IN_MERGE_REQUEST
	NOTE_UNKNOWN
)

func (nt NoteType) String() string {
	switch nt {
	case NOTE_COMMENT:
		return "note comment"
	case NOTE_TITLE_CHANGED:
		return "note title changed"
	case NOTE_DESCRIPTION_CHANGED:
		return "note description changed"
	case NOTE_CLOSED:
		return "note closed"
	case NOTE_REOPENED:
		return "note reopened"
	case NOTE_LOCKED:
		return "note locked"
	case NOTE_UNLOCKED:
		return "note unlocked"
	case NOTE_CHANGED_DUEDATE:
		return "note changed duedate"
	case NOTE_REMOVED_DUEDATE:
		return "note remove duedate"
	case NOTE_ASSIGNED:
		return "note assigned"
	case NOTE_UNASSIGNED:
		return "note unassigned"
	case NOTE_CHANGED_MILESTONE:
		return "note changed milestone"
	case NOTE_REMOVED_MILESTONE:
		return "note removed in milestone"
	case NOTE_MENTIONED_IN_ISSUE:
		return "note mentioned in issue"
	case NOTE_MENTIONED_IN_MERGE_REQUEST:
		return "note mentioned in merge request"
	case NOTE_UNKNOWN:
		return "note unknown"
	default:
		panic("unknown note type")
	}
}

// GetNoteType parse a note system and body and return the note type and it content
func GetNoteType(n *gitlab.Note) (NoteType, string) {
	// when a note is a comment system is set to false
	// when a note is a different event system is set to true
	// because Gitlab
	if !n.System {
		return NOTE_COMMENT, n.Body
	}

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

	if strings.HasPrefix(n.Body, "mentioned in issue") {
		return NOTE_MENTIONED_IN_ISSUE, ""
	}

	if strings.HasPrefix(n.Body, "mentioned in merge request") {
		return NOTE_MENTIONED_IN_MERGE_REQUEST, ""
	}

	return NOTE_UNKNOWN, ""
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
