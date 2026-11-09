package config

import (
	"github.com/git-bug/git-bug/entity/dag"
)

var _ dag.Snapshot = &Snapshot{}

// Snapshot is a compiled form of the Config data structure, made of its sections.
// A setting that was never set reads as its default.
type Snapshot struct {
	Repo   RepoSection
	Labels LabelsSection

	Operations []dag.Operation
}

func (snap *Snapshot) AllOperations() []dag.Operation {
	return snap.Operations
}

func (snap *Snapshot) AppendOperation(op dag.Operation) {
	snap.Operations = append(snap.Operations, op)
}
