package bug

import (
	"github.com/MichaelMure/git-bug/repository"
)

var _ Interface = &WithSnapshot{}

// WithSnapshot encapsulate a Bug and maintain the corresponding Snapshot efficiently
type WithSnapshot struct {
	*Bug
	snap *Snapshot
}

func (b *WithSnapshot) Compile() *Snapshot {
	if b.snap == nil {
		snap := b.Bug.Compile()
		b.snap = snap
	}
	return b.snap
}

// Append intercept Bug.Append() to update the snapshot efficiently
func (b *WithSnapshot) Append(op Operation) {
	b.Bug.Append(op)

	if b.snap == nil {
		return
	}

	op.Apply(b.snap)
	b.snap.Operations = append(b.snap.Operations, op)
}

// Commit intercept Bug.Commit() to update the snapshot efficiently
func (b *WithSnapshot) Commit(repo repository.ClockedRepo) error {
	err := b.Bug.Commit(repo)

	if err != nil {
		b.snap = nil
		return err
	}

	// Commit() shouldn't change anything of the bug state apart from the
	// initial ID set

	if b.snap == nil {
		return nil
	}

	b.snap.id = b.Bug.Id()
	return nil
}
