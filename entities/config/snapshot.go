package config

import (
	"maps"
	"slices"
	"sync"

	"github.com/git-bug/git-bug/entity/dag"
	"github.com/git-bug/git-bug/util/lamport"
)

var _ dag.Snapshot = &Snapshot{}

// Settings is the config at one point of its history, made of its sections. A
// setting that was never set reads as its default.
type Settings struct {
	Repo   RepoSection
	Labels LabelsSection
}

// Snapshot is a compiled form of the Config data structure: the latest settings,
// and the operations in effect.
type Snapshot struct {
	Settings // latest version

	Operations []dag.Operation

	// mu guards eras
	mu sync.Mutex
	// eras holds the eras of each clock asked to At, built from Operations on
	// first use
	eras map[string]*eras
}

func (snap *Snapshot) AllOperations() []dag.Operation {
	return snap.Operations
}

func (snap *Snapshot) AppendOperation(op dag.Operation) {
	snap.Operations = append(snap.Operations, op)

	snap.mu.Lock()
	defer snap.mu.Unlock()
	snap.eras = nil
}

// Clone returns a copy of the snapshot, to apply operations to without changing
// the original. The copy builds its own eras.
func (snap *Snapshot) Clone() *Snapshot {
	return &Snapshot{
		Settings: Settings{
			Repo: snap.Repo,
			Labels: LabelsSection{
				definitions: maps.Clone(snap.Labels.definitions),
				allowed:     maps.Clone(snap.Labels.allowed),
			},
		},
		Operations: slices.Clone(snap.Operations),
	}
}

// An operation of another entity is judged by the config as it was when that
// operation was made, not by the latest one. Each config operation records the
// time of every clock of the repository when it was made, see opBase. It applies
// to the operations of a later time on the clock of their namespace: an operation
// of the same time or earlier was made before it, and is judged without it. A clock
// that an operation didn't record counts as 0, the operation applying at any time.
//
// As for any operation, this order only holds as far as the authors' clocks knew:
// reading a config doesn't raise the clocks of the other entities. A client that
// has a config without the operations its author had sees them as concurrent, and
// may make an operation judged by an era before that config operation. Every client
// judges it the same way.
//
// Along one clock, the times recorded by the config operations cut the timeline
// into eras: the config is the same for every time of an era. The first era comes
// before any recorded time, with no config operation: every setting reads as its
// default.

// At returns the settings judging an operation of another entity of lamport time t
// on the clock clockName: the ones of the era holding t. The operations of an
// entity are judged on the edit clock of its namespace, by the edit time of their
// pack.
//
// The eras of a clock are built on the first call for it, and kept with the
// snapshot.
func (snap *Snapshot) At(clockName string, time lamport.Time) Settings {
	snap.mu.Lock()
	defer snap.mu.Unlock()

	e, ok := snap.eras[clockName]
	if !ok {
		e = newEras(snap.Operations, clockName)
		if snap.eras == nil {
			snap.eras = make(map[string]*eras)
		}
		snap.eras[clockName] = e
	}
	return e.at(time)
}

// eras are the settings of the eras of one clock.
type eras struct {
	// bounds are the distinct times of the clock recorded by the operations, in
	// increasing order. Era i holds the times above bounds[i-1] and up to bounds[i],
	// the last era the times above the last bound.
	bounds []lamport.Time
	// settings holds the settings of each era, one more than bounds
	settings []Settings
}

// newEras returns the eras of the clock clockName, made by the operations of a
// snapshot. Those are the operations in effect, decided once in the order of the
// config: an era applies the ones recorded before it, in that order.
func newEras(operations []dag.Operation, clockName string) *eras {
	ops := make([]Operation, len(operations))
	stamps := make([]lamport.Time, len(operations))
	for i, op := range operations {
		ops[i] = op.(Operation)
		stamps[i] = opTimes(op)[clockName]
	}

	bounds := slices.Compact(slices.Sorted(slices.Values(stamps)))

	// each era replays the operations: there are few of them in a config
	settings := make([]Settings, len(bounds)+1)
	for i := range settings {
		era := &Snapshot{}
		for j, op := range ops {
			if i > 0 && stamps[j] <= bounds[i-1] {
				op.Apply(era)
			}
		}
		settings[i] = era.Settings
	}

	return &eras{bounds: bounds, settings: settings}
}

// at returns the settings of the era holding a time.
func (e *eras) at(time lamport.Time) Settings {
	i, _ := slices.BinarySearch(e.bounds, time)
	return e.settings[i]
}
