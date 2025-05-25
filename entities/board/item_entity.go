package board

import (
	"github.com/git-bug/git-bug/entities/bug"
	"github.com/git-bug/git-bug/entity"
)

var _ Item = &BugItem{}

type BugItem struct {
	combinedId entity.CombinedId
	Bug        bug.Interface
}

func (e *BugItem) CombinedId() entity.CombinedId {
	if e.combinedId == "" || e.combinedId == entity.UnsetCombinedId {
		// simply panic as it would be a coding error (no id provided at construction)
		panic("no combined id")
	}
	return e.combinedId
}
