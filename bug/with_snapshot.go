package bug

import "github.com/MichaelMure/git-bug/repository"

var _ Interface = &WithSnapshot{}

// WithSnapshot encapsulate a Bug and maintain the corresponding Snapshot efficiently
type WithSnapshot struct {
	*Bug
	snap *Snapshot
}

// Snapshot return the current snapshot
func (b *WithSnapshot) Snapshot() *Snapshot {
	if b.snap == nil {
		snap := b.Bug.Compile()
		b.snap = &snap
	}
	return b.snap
}

// Append intercept Bug.Append() to update the snapshot efficiently
func (b *WithSnapshot) Append(op Operation) {
	b.Bug.Append(op)

	if b.snap == nil {
		return
	}

	snap := op.Apply(*b.snap)
	snap.Operations = append(snap.Operations, op)

	b.snap = &snap
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

	b.snap.id = b.Bug.id
	return nil
}

// Merge intercept Bug.Merge() and clear the snapshot
func (b *WithSnapshot) Merge(repo repository.Repo, other Interface) (bool, error) {
	b.snap = nil
	return b.Bug.Merge(repo, other)
}
