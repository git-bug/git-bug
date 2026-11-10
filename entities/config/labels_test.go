package config

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/entities/common"
	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/entity/dag/dagtest"
)

func TestLabels(t *testing.T) {
	clocks, rene, isaac := makeTestContext(t)
	unix := time.Now().Unix()
	red, blue := "#f44336", "#2196f3"

	must := func(op Operation, err error) Operation {
		require.NoError(t, err)
		return op
	}

	create := func(t *testing.T) *Config {
		c, _, err := Create(clocks, rene, unix)
		require.NoError(t, err)
		return c
	}

	t.Run("validate", func(t *testing.T) {
		// lengths are counted in characters: "é" is two bytes
		longestName := common.Label(strings.Repeat("é", 64))
		longestDescription := strings.Repeat("é", 256)

		good := []Operation{
			must(NewLabelsSetOp(clocks, rene, unix, longestName, LabelsDefinition{Description: SetField(longestDescription)})),
			must(NewLabelsAllowOp(clocks, rene, unix, longestName)),
			must(NewLabelsSetOp(clocks, rene, unix, "bug", LabelsDefinition{})),
			must(NewLabelsSetOp(clocks, rene, unix, "bug", LabelsDefinition{Color: SetField(red), Description: SetField("something is broken")})),
			must(NewLabelsSetOp(clocks, rene, unix, "bug", LabelsDefinition{Color: ClearField[string](), Description: ClearField[string]()})),
			must(NewLabelsRemoveOp(clocks, rene, unix, "bug")),
			must(NewLabelsAllowOp(clocks, rene, unix, "bug")),
			must(NewLabelsDisallowOp(clocks, rene, unix, "bug")),
		}
		for i, op := range good {
			require.NoError(t, op.Validate(), "operation %d", i)
		}

		bad := []Operation{
			must(NewLabelsSetOp(clocks, rene, unix, longestName+"a", LabelsDefinition{})),
			must(NewLabelsSetOp(clocks, rene, unix, "bug", LabelsDefinition{Description: SetField(longestDescription + "a")})),
			must(NewLabelsRemoveOp(clocks, rene, unix, longestName+"a")),
			must(NewLabelsAllowOp(clocks, rene, unix, longestName+"a")),
			must(NewLabelsSetOp(clocks, rene, unix, "", LabelsDefinition{})),
			must(NewLabelsSetOp(clocks, rene, unix, "multi\nline", LabelsDefinition{})),
			must(NewLabelsSetOp(clocks, rene, unix, "bug", LabelsDefinition{Color: SetField("red")})),
			must(NewLabelsSetOp(clocks, rene, unix, "bug", LabelsDefinition{Color: SetField("#F44336")})),
			must(NewLabelsSetOp(clocks, rene, unix, "bug", LabelsDefinition{Color: SetField("#f4433")})),
			must(NewLabelsSetOp(clocks, rene, unix, "bug", LabelsDefinition{Color: SetField("#f4433g")})),
			must(NewLabelsSetOp(clocks, rene, unix, "bug", LabelsDefinition{Description: SetField("")})),
			must(NewLabelsSetOp(clocks, rene, unix, "bug", LabelsDefinition{Description: SetField("multi\nline")})),
			must(NewLabelsRemoveOp(clocks, rene, unix, "")),
			must(NewLabelsAllowOp(clocks, rene, unix, "")),
			must(NewLabelsDisallowOp(clocks, rene, unix, "multi\nline")),
		}
		for i, op := range bad {
			require.Error(t, op.Validate(), "operation %d", i)
		}
	})

	t.Run("serialize", func(t *testing.T) {
		definitions := []LabelsDefinition{
			{},
			{Color: SetField(red), Description: SetField("something is broken")},
			{Color: SetField(red), Description: ClearField[string]()},
		}
		for _, definition := range definitions {
			dagtest.SerializeRoundTripTest(t, operationUnmarshaler, func(author identity.Interface, unixTime int64) (*LabelsSetOperation, entity.Resolvers) {
				op, err := NewLabelsSetOp(clocks, author, unixTime, "bug", definition)
				require.NoError(t, err)
				return op, nil
			})
		}
		dagtest.SerializeRoundTripTest(t, operationUnmarshaler, func(author identity.Interface, unixTime int64) (*LabelsRemoveOperation, entity.Resolvers) {
			op, err := NewLabelsRemoveOp(clocks, author, unixTime, "bug")
			require.NoError(t, err)
			return op, nil
		})
		dagtest.SerializeRoundTripTest(t, operationUnmarshaler, func(author identity.Interface, unixTime int64) (*LabelsAllowOperation, entity.Resolvers) {
			op, err := NewLabelsAllowOp(clocks, author, unixTime, "bug")
			require.NoError(t, err)
			return op, nil
		})
		dagtest.SerializeRoundTripTest(t, operationUnmarshaler, func(author identity.Interface, unixTime int64) (*LabelsDisallowOperation, entity.Resolvers) {
			op, err := NewLabelsDisallowOp(clocks, author, unixTime, "bug")
			require.NoError(t, err)
			return op, nil
		})
	})

	t.Run("json", func(t *testing.T) {
		// fieldsOf returns the "fields" of the JSON of an operation, false if absent
		fieldsOf := func(t *testing.T, op Operation) (string, bool) {
			data, err := json.Marshal(op)
			require.NoError(t, err)
			var raw map[string]json.RawMessage
			require.NoError(t, json.Unmarshal(data, &raw))
			fields, ok := raw["fields"]
			return string(fields), ok
		}

		// a field set is written with its value, a cleared one as null, an absent
		// one not at all
		fields, ok := fieldsOf(t, must(NewLabelsSetOp(clocks, rene, unix, "crash", LabelsDefinition{Color: SetField(red), Description: ClearField[string]()})))
		require.True(t, ok)
		require.JSONEq(t, `{"color": "#f44336", "description": null}`, fields)

		fields, ok = fieldsOf(t, must(NewLabelsSetOp(clocks, rene, unix, "crash", LabelsDefinition{Color: SetField(red)})))
		require.True(t, ok)
		require.JSONEq(t, `{"color": "#f44336"}`, fields)

		// with no field, "fields" is left out
		_, ok = fieldsOf(t, must(NewLabelsSetOp(clocks, rene, unix, "crash", LabelsDefinition{})))
		require.False(t, ok)
		_, ok = fieldsOf(t, must(NewLabelsAllowOp(clocks, rene, unix, "crash")))
		require.False(t, ok)

		// a field this client doesn't know is ignored, and the others apply
		raw := `{"type": 20001, "timestamp": 1609459200, "nonce": "AAAAAAAAAAAAAAAAAAAAAAAAAAA=", "key": "crash", "fields": {"color": "#f44336", "priority": 1}}`
		op, err := operationUnmarshaler(t.Context(), json.RawMessage(raw), nil)
		require.NoError(t, err)
		require.Equal(t, LabelsDefinition{Color: SetField(red)}, op.(*LabelsSetOperation).Fields)
	})

	t.Run("definitions", func(t *testing.T) {
		c := create(t)
		define := func(name common.Label, change LabelsDefinition) func() {
			return func() {
				_, err := LabelsSet(c, clocks, rene, unix, name, change)
				require.NoError(t, err)
			}
		}
		remove := func(name common.Label) func() {
			return func() {
				_, err := LabelsRemove(c, clocks, rene, unix, name)
				require.NoError(t, err)
			}
		}

		// the example of the spec, then a change of one field at a time
		steps := []struct {
			name string
			do   func()
			want map[common.Label]LabelsDefinition
		}{
			{
				name: "define",
				do:   define("crash", LabelsDefinition{Color: SetField(red)}),
				want: map[common.Label]LabelsDefinition{
					"crash": {Color: SetField(red)},
				},
			},
			{
				name: "set another field",
				do:   define("crash", LabelsDefinition{Description: SetField("Program dies")}),
				want: map[common.Label]LabelsDefinition{
					"crash": {Color: SetField(red), Description: SetField("Program dies")},
				},
			},
			{
				name: "define with no field",
				do:   define("regression", LabelsDefinition{}),
				want: map[common.Label]LabelsDefinition{
					"crash":      {Color: SetField(red), Description: SetField("Program dies")},
					"regression": {},
				},
			},
			{
				name: "remove",
				do:   remove("crash"),
				want: map[common.Label]LabelsDefinition{
					"regression": {},
				},
			},
			{
				name: "define again, with no field back",
				do:   define("crash", LabelsDefinition{Color: SetField(blue)}),
				want: map[common.Label]LabelsDefinition{
					"crash":      {Color: SetField(blue)},
					"regression": {},
				},
			},
			{
				name: "change one field, set another",
				do:   define("crash", LabelsDefinition{Color: SetField(red), Description: SetField("Program exits")}),
				want: map[common.Label]LabelsDefinition{
					"crash":      {Color: SetField(red), Description: SetField("Program exits")},
					"regression": {},
				},
			},
			{
				name: "clear a field",
				do:   define("crash", LabelsDefinition{Color: ClearField[string]()}),
				want: map[common.Label]LabelsDefinition{
					"crash":      {Description: SetField("Program exits")},
					"regression": {},
				},
			},
		}
		for _, step := range steps {
			step.do()
			require.Equal(t, map_[common.Label, LabelsDefinition](step.want), c.Compile().Labels.definitions, step.name)
		}

		snap := c.Compile()
		require.Equal(t, []common.Label{"crash", "regression"}, snap.Labels.Names())
		definition, ok := snap.Labels.Get("crash")
		require.True(t, ok)
		description, ok := definition.Description.Get()
		require.True(t, ok)
		require.Equal(t, "Program exits", description)
		_, ok = snap.Labels.Get("wontfix")
		require.False(t, ok)

		// a color set is read from its hex digits, otherwise it is derived from the name
		define("regression", LabelsDefinition{Color: SetField(blue)})()
		snap = c.Compile()
		require.Equal(t, common.LabelColor{R: 0x21, G: 0x96, B: 0xf3, A: 255}, snap.Labels.Color("regression"))
		require.Equal(t, common.Label("crash").Color(), snap.Labels.Color("crash"))
		require.Equal(t, common.Label("wontfix").Color(), snap.Labels.Color("wontfix"))
	})

	t.Run("last writer wins", func(t *testing.T) {
		c := create(t)
		_, err := LabelsSet(c, clocks, rene, unix, "bug", LabelsDefinition{Color: SetField(red)})
		require.NoError(t, err)
		_, err = LabelsSet(c, clocks, isaac, unix, "bug", LabelsDefinition{Color: SetField(blue)})
		require.NoError(t, err)

		definition, ok := c.Compile().Labels.Get("bug")
		require.True(t, ok)
		require.Equal(t, LabelsDefinition{Color: SetField(blue)}, definition)
	})

	t.Run("allowed", func(t *testing.T) {
		c := create(t)
		_, err := LabelsSet(c, clocks, rene, unix, "bug", LabelsDefinition{Color: SetField(red)})
		require.NoError(t, err)

		// any label is allowed until one is
		snap := c.Compile()
		require.True(t, snap.Labels.Allowed("bug"))
		require.True(t, snap.Labels.Allowed("anything"))
		require.Empty(t, snap.Labels.AllowedNames())

		_, err = LabelsAllow(c, clocks, rene, unix, "bug")
		require.NoError(t, err)
		_, err = LabelsAllow(c, clocks, rene, unix, "feature")
		require.NoError(t, err)

		snap = c.Compile()
		require.True(t, snap.Labels.Allowed("bug"))
		require.True(t, snap.Labels.Allowed("feature"))
		require.False(t, snap.Labels.Allowed("anything"))
		require.Equal(t, []common.Label{"bug", "feature"}, snap.Labels.AllowedNames())
		// allowing a label doesn't define it
		require.Equal(t, []common.Label{"bug"}, snap.Labels.Names())

		// disallowing keeps the definition
		_, err = LabelsDisallow(c, clocks, rene, unix, "bug")
		require.NoError(t, err)

		snap = c.Compile()
		require.False(t, snap.Labels.Allowed("bug"))
		require.Equal(t, []common.Label{"feature"}, snap.Labels.AllowedNames())
		require.Equal(t, []common.Label{"bug"}, snap.Labels.Names())
		require.Equal(t, common.LabelColor{R: 0xf4, G: 0x43, B: 0x36, A: 255}, snap.Labels.Color("bug"))

		// disallowing the last allowed label allows any label again
		_, err = LabelsDisallow(c, clocks, rene, unix, "feature")
		require.NoError(t, err)

		snap = c.Compile()
		require.True(t, snap.Labels.Allowed("anything"))
		require.Empty(t, snap.Labels.AllowedNames())
	})

	t.Run("restrict and lift", func(t *testing.T) {
		c := create(t)

		_, err := LabelsRestrict(c, clocks, rene, unix)
		require.Error(t, err)
		_, err = LabelsLiftRestriction(c, clocks, rene, unix)
		require.Error(t, err)

		_, err = LabelsSet(c, clocks, rene, unix, "bug", LabelsDefinition{})
		require.NoError(t, err)
		_, err = LabelsSet(c, clocks, rene, unix, "feature", LabelsDefinition{})
		require.NoError(t, err)

		allowed, err := LabelsRestrict(c, clocks, rene, unix)
		require.NoError(t, err)
		require.Len(t, allowed, 2)

		snap := c.Compile()
		require.Equal(t, []common.Label{"bug", "feature"}, snap.Labels.AllowedNames())
		require.False(t, snap.Labels.Allowed("anything"))

		disallowed, err := LabelsLiftRestriction(c, clocks, rene, unix)
		require.NoError(t, err)
		require.Len(t, disallowed, 2)

		snap = c.Compile()
		require.Empty(t, snap.Labels.AllowedNames())
		require.True(t, snap.Labels.Allowed("anything"))
		require.Equal(t, []common.Label{"bug", "feature"}, snap.Labels.Names())
	})
}
