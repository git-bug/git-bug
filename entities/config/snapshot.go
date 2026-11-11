package config

import (
	"maps"
	"slices"
	"sync"

	"github.com/git-bug/git-bug/entity"
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
	e := snap.erasOf(clockName)
	return e.settings[e.index(time)]
}

// EraId returns the id of the era holding a lamport time on a clock, see At. Two
// eras with the same id have the same settings, made of the same operations: for
// an entity judged by the config, the id at the edit time of its latest operation
// tells whether it is judged as before, see ForOperation. The first era, with no
// operation, has an empty id, as has a repository without a config.
func (snap *Snapshot) EraId(clockName string, time lamport.Time) string {
	e := snap.erasOf(clockName)
	return e.ids[e.index(time)]
}

// ForOperation returns the settings judging an operation of another entity, of
// the namespace namespace: the ones of the era of its edit time, on the edit clock
// of the namespace, see At.
//
// An operation not committed yet has no edit time: it is judged by the latest
// settings, the ones its author sees. Once committed, it is judged by the era of
// its edit time, an earlier one if a config operation recorded a later time than
// this client's clock, see At.
func (snap *Snapshot) ForOperation(namespace string, op dag.Operation) Settings {
	if op.EditTime() == 0 {
		return snap.Settings
	}
	return snap.At(dag.EditClock(namespace), op.EditTime())
}

// erasOf returns the eras of a clock, built on the first call for it.
func (snap *Snapshot) erasOf(clockName string) *eras {
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
	return e
}

// eras are the settings of the eras of one clock.
type eras struct {
	// bounds are the distinct times of the clock recorded by the operations, in
	// increasing order. Era i holds the times above bounds[i-1] and up to bounds[i],
	// the last era the times above the last bound.
	bounds []lamport.Time
	// settings holds the settings of each era, one more than bounds
	settings []Settings
	// ids holds the id of each era, see EraId
	ids []string
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
	ids := make([]string, len(bounds)+1)
	for i := range settings {
		era := &Snapshot{}
		// the ids of the operations applied, in order
		var applied []byte
		for j, op := range ops {
			if i > 0 && stamps[j] <= bounds[i-1] {
				op.Apply(era)
				applied = append(applied, op.Id()...)
			}
		}
		settings[i] = era.Settings
		if len(applied) > 0 {
			ids[i] = entity.DeriveId(applied).String()
		}
	}

	return &eras{bounds: bounds, settings: settings, ids: ids}
}

// index returns the index of the era holding a time.
func (e *eras) index(time lamport.Time) int {
	i, _ := slices.BinarySearch(e.bounds, time)
	return i
}
