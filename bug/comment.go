package bug

import (
	"strings"

	"github.com/dustin/go-humanize"

	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/identity"
	"github.com/MichaelMure/git-bug/repository"
	"github.com/MichaelMure/git-bug/util/timestamp"
)

// Comment represent a comment in a Bug
type Comment struct {
	id      entity.Id
	Author  identity.Interface
	Message string
	Files   []repository.Hash

	// Creation time of the comment.
	// Should be used only for human display, never for ordering as we can't rely on it in a distributed system.
	UnixTime timestamp.Timestamp
}

// Id return the Comment identifier
func (c Comment) Id() entity.Id {
	if c.id == "" {
		// simply panic as it would be a coding error
		// (using an id of an identity not stored yet)
		panic("no id yet")
	}
	return c.id
}

const compiledCommentIdFormat = "BCBCBCBBBCBBBBCBBBBCBBBBCBBBBCBBBBCBBBBC"

// DeriveCommentId compute a merged Id for a comment holding information from
// both the Bug's Id and the Comment's Id. This allow to later find efficiently
// a Comment because we can access the bug directly instead of searching for a
// Bug that has a Comment matching the Id.
//
// To allow the use of an arbitrary length prefix of this merged Id, Ids from Bug
// and Comment are interleaved with this irregular pattern to give the best chance
// to find the Comment even with a 7 character prefix.
//
// A complete merged Id hold 30 characters for the Bug and 10 for the Comment,
// which give a key space of 36^30 for the Bug (~5 * 10^46) and 36^10 for the
// Comment (~3 * 10^15). This asymmetry assume a reasonable number of Comment
// within a Bug, while still allowing for a vast key space for Bug (that is, a
// globally merged bug database) with a low risk of collision.
func DeriveCommentId(bugId entity.Id, commentId entity.Id) entity.Id {
	var id strings.Builder
	for _, char := range compiledCommentIdFormat {
		if char == 'B' {
			id.WriteByte(bugId[0])
			bugId = bugId[1:]
		} else {
			id.WriteByte(commentId[0])
			commentId = commentId[1:]
		}
	}
	return entity.Id(id.String())
}

func SplitCommentId(prefix string) (bugPrefix string, commentPrefix string) {
	var bugIdPrefix strings.Builder
	var commentIdPrefix strings.Builder

	for i, char := range prefix {
		if compiledCommentIdFormat[i] == 'B' {
			bugIdPrefix.WriteRune(char)
		} else {
			commentIdPrefix.WriteRune(char)
		}
	}
	return bugIdPrefix.String(), commentIdPrefix.String()
}

// FormatTimeRel format the UnixTime of the comment for human consumption
func (c Comment) FormatTimeRel() string {
	return humanize.Time(c.UnixTime.Time())
}

func (c Comment) FormatTime() string {
	return c.UnixTime.Time().Format("Mon Jan 2 15:04:05 2006 +0200")
}

// Sign post method for gqlgen
func (c Comment) IsAuthored() {}
