package board

import (
	"github.com/git-bug/git-bug/entities/bug"
	"github.com/git-bug/git-bug/entities/common"
	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/entity/dag"
)

var _ Item = &BugItem{}

type BugItem struct {
	combinedId entity.CombinedId
	Bug        dag.CompileTo[*bug.Snapshot]
}

func (e *BugItem) CombinedId() entity.CombinedId {
	if e.combinedId == "" || e.combinedId == entity.UnsetCombinedId {
		// simply panic as it would be a coding error (no id provided at construction)
		panic("no combined id")
	}
	return e.combinedId
}

func (e *BugItem) Author() identity.Interface {
	return e.Bug.Snapshot().Author
}

func (e *BugItem) Title() string {
	return e.Bug.Snapshot().Title
}

func (e *BugItem) Labels() []common.Label {
	return e.Bug.Snapshot().Labels
}

// IsAuthored is a sign post method for gqlgen
func (e *BugItem) IsAuthored() {}
