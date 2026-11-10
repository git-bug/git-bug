package config

import (
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/entities/common"
	"github.com/git-bug/git-bug/entity/dag"
	"github.com/git-bug/git-bug/repository"
	"github.com/git-bug/git-bug/util/lamport"
)

func TestSnapshot(t *testing.T) {
	clocks, rene, _ := makeTestContext(t)
	unix := time.Now().Unix()

	t.Run("clone", func(t *testing.T) {
		red := common.LabelColor{R: 0xf4, G: 0x43, B: 0x36, A: 255}
		blue := common.LabelColor{R: 0x21, G: 0x96, B: 0xf3, A: 255}

		c, _, err := Create(clocks, rene, unix)
		require.NoError(t, err)
		_, err = LabelsSet(c, clocks, rene, unix, "bug", LabelsDefinition{Color: SetField("#f44336")})
		require.NoError(t, err)
		_, err = LabelsAllow(c, clocks, rene, unix, "bug")
		require.NoError(t, err)
		snap := c.Compile()
		era := snap.At("bugs-edit", 43)

		// operations applied to the clone leave the original as it was
		clone := snap.Clone()
		set, err := NewLabelsSetOp(clocks, rene, unix, "bug", LabelsDefinition{Color: SetField("#2196f3")})
		require.NoError(t, err)
		allow, err := NewLabelsAllowOp(clocks, rene, unix, "feature")
		require.NoError(t, err)
		description, err := NewRepoSetDescriptionOp(clocks, rene, unix, "description")
		require.NoError(t, err)
		for _, op := range []Operation{set, allow, description} {
			op.Apply(clone)
			clone.AppendOperation(op)
		}

		require.Equal(t, red, snap.Labels.Color("bug"))
		require.Equal(t, []common.Label{"bug"}, snap.Labels.AllowedNames())
		require.Empty(t, snap.Repo.Description())
		require.Len(t, snap.Operations, 3)
		require.Same(t, era, snap.At("bugs-edit", 43))

		// the clone builds its own eras
		require.Equal(t, blue, clone.Labels.Color("bug"))
		require.Equal(t, []common.Label{"bug", "feature"}, clone.Labels.AllowedNames())
		require.Equal(t, "description", clone.At("bugs-edit", 43).Repo.Description())
	})
}

func TestEras(t *testing.T) {
	unix := time.Now().Unix()

	t.Run("eras", func(t *testing.T) {
		// the operations record bugs-edit at 42, then 50, then 60
		clocks, rene, _ := makeTestContext(t)
		c, _, err := Create(clocks, rene, unix)
		require.NoError(t, err)
		_, err = LabelsSet(c, clocks, rene, unix, "bug", LabelsDefinition{Color: SetField("#f44336")})
		require.NoError(t, err)
		require.NoError(t, clocks.Witness("bugs-edit", 50))
		_, err = RepoSetDescription(c, clocks, rene, unix, "description")
		require.NoError(t, err)
		_, err = LabelsAllow(c, clocks, rene, unix, "bug")
		require.NoError(t, err)
		require.NoError(t, clocks.Witness("bugs-edit", 60))
		_, err = LabelsSet(c, clocks, rene, unix, "bug", LabelsDefinition{Color: SetField("#2196f3")})
		require.NoError(t, err)

		snap := c.Compile()
		at := func(time lamport.Time) *Snapshot { return snap.At("bugs-edit", time) }

		red := common.LabelColor{R: 0xf4, G: 0x43, B: 0x36, A: 255}
		blue := common.LabelColor{R: 0x21, G: 0x96, B: 0xf3, A: 255}
		cases := []struct {
			times       []lamport.Time
			description string
			color       common.LabelColor
			restricted  bool
			operations  int
		}{
			// before the config, every setting has its default
			{times: []lamport.Time{0, 1, 42}, color: common.Label("bug").Color(), operations: 0},
			// an operation applies strictly after the time it recorded
			{times: []lamport.Time{43, 50}, color: red, operations: 2},
			{times: []lamport.Time{51, 60}, description: "description", color: red, restricted: true, operations: 4},
			{times: []lamport.Time{61, 1000}, description: "description", color: blue, restricted: true, operations: 5},
		}
		for _, want := range cases {
			for _, time := range want.times {
				era := at(time)
				require.Equal(t, want.description, era.Repo.Description(), "time %d", time)
				require.Equal(t, want.color, era.Labels.Color("bug"), "time %d", time)
				require.Equal(t, want.restricted, !era.Labels.Allowed("anything"), "time %d", time)
				require.Len(t, era.Operations, want.operations, "time %d", time)
			}
		}

		// the last era is the latest config
		require.Equal(t, snap.Repo, at(61).Repo)
		require.Equal(t, snap.Labels, at(61).Labels)
	})

	t.Run("kept with the snapshot", func(t *testing.T) {
		clocks, rene, _ := makeTestContext(t)
		c, _, err := Create(clocks, rene, unix)
		require.NoError(t, err)
		snap := c.Compile()

		// built once, from any goroutine
		eras := make([]*Snapshot, 8)
		var wg sync.WaitGroup
		for i := range eras {
			wg.Go(func() { eras[i] = snap.At("bugs-edit", 43) })
		}
		wg.Wait()
		for _, era := range eras {
			require.Same(t, eras[0], era)
		}
		// the times of an era share its config
		require.Same(t, eras[0], snap.At("bugs-edit", 1000))

		// appending an operation builds them again
		op, err := NewRepoSetDescriptionOp(clocks, rene, unix, "description")
		require.NoError(t, err)
		op.Apply(snap)
		snap.AppendOperation(op)
		require.Equal(t, "description", snap.At("bugs-edit", 43).Repo.Description())
	})

	t.Run("an earlier time sorted later", func(t *testing.T) {
		// an operation made in a clone whose bugs were behind, merged after the others
		clocks, rene, _ := makeTestContext(t)
		behind := repository.NewMockRepoClock()
		_, err := behind.GetOrCreateClock("bugs-edit", 10)
		require.NoError(t, err)

		c, _, err := Create(clocks, rene, unix)
		require.NoError(t, err)
		_, err = RepoSetDescription(c, clocks, rene, unix, "current")
		require.NoError(t, err)
		_, err = RepoSetDescription(c, behind, rene, unix, "behind")
		require.NoError(t, err)

		snap := c.Compile()
		require.Empty(t, snap.At("bugs-edit", 10).Repo.Description())
		// it applies alone from its own time, and wins where both apply, as it sorts
		// last in the config
		require.Equal(t, "behind", snap.At("bugs-edit", 11).Repo.Description())
		require.Len(t, snap.At("bugs-edit", 11).Operations, 1)
		require.Equal(t, "behind", snap.At("bugs-edit", 43).Repo.Description())
		require.Len(t, snap.At("bugs-edit", 43).Operations, 3)
	})

	t.Run("unrecorded clock", func(t *testing.T) {
		// a clock that didn't exist when the operations were made counts as 0
		clocks, rene, _ := makeTestContext(t)
		c, _, err := Create(clocks, rene, unix)
		require.NoError(t, err)
		_, err = RepoSetDescription(c, clocks, rene, unix, "description")
		require.NoError(t, err)

		snap := c.Compile()
		require.Empty(t, snap.At("prs-edit", 0).Repo.Description())
		require.Equal(t, "description", snap.At("prs-edit", 1).Repo.Description())
	})

	t.Run("unknown operation", func(t *testing.T) {
		// an operation this client doesn't know is placed by the times of its JSON
		clocks, rene, _ := makeTestContext(t)
		c, _, err := Create(clocks, rene, unix)
		require.NoError(t, err)

		raw, err := json.Marshal(map[string]any{
			"type":      999,
			"timestamp": unix,
			"nonce":     make([]byte, 20),
			"times":     map[string]lamport.Time{"bugs-edit": 55},
		})
		require.NoError(t, err)
		unknown, err := operationUnmarshaler(t.Context(), raw, nil)
		require.NoError(t, err)
		require.IsType(t, &dag.UnknownOperation[*Snapshot]{}, unknown)
		c.Append(unknown.(Operation))

		snap := c.Compile()
		require.Len(t, snap.At("bugs-edit", 55).Operations, 1)
		require.Len(t, snap.At("bugs-edit", 56).Operations, 2)
		require.Same(t, unknown, snap.At("bugs-edit", 56).Operations[1])
	})
}
