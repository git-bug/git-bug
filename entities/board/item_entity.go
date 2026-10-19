package board

import (
	"github.com/git-bug/git-bug/entity"
)

var _ Item = &BugItem{}

// BugItem is a board item referring to a bug. It only holds the bug's id: the
// bug itself lives on its own and is resolved when the board is displayed.
type BugItem struct {
	combinedId entity.CombinedId
	BugId      entity.Id
}

func (e *BugItem) CombinedId() entity.CombinedId {
	if e.combinedId == "" || e.combinedId == entity.UnsetCombinedId {
		// simply panic as it would be a coding error (no id provided at construction)
		panic("no combined id")
	}
	return e.combinedId
}
