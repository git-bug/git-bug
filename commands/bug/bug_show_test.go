package bugcmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/commands/bug/testenv"
)

func TestBugShowJsonSingleIDRemainsObject(t *testing.T) {
	env, bugID := testenv.NewTestEnvAndBug(t)

	opts := bugShowOptions{format: "json"}
	require.NoError(t, runBugShow(env, opts, []string{bugID.Human()}))

	require.True(t, strings.HasPrefix(env.Out.String(), "{"))

	var got map[string]any
	require.NoError(t, json.Unmarshal(env.Out.Bytes(), &got))
	require.Equal(t, bugID.String(), got["id"])
}

func TestBugShowJsonMultipleIDsIsArrayInArgumentOrder(t *testing.T) {
	env, b1ID := testenv.NewTestEnvAndBug(t)
	b2, _, err := env.Backend.Bugs().New("second bug title", "second bug body")
	require.NoError(t, err)

	opts := bugShowOptions{format: "json"}
	require.NoError(t, runBugShow(env, opts, []string{b2.Id().Human(), b1ID.Human()}))

	var got []map[string]any
	require.NoError(t, json.Unmarshal(env.Out.Bytes(), &got))
	require.Len(t, got, 2)
	require.Equal(t, b2.Id().Human(), got[0]["human_id"])
	require.Equal(t, b1ID.Human(), got[1]["human_id"])
}

func TestBugShowMultipleIDsRejectDefaultFormat(t *testing.T) {
	env, b1ID := testenv.NewTestEnvAndBug(t)
	b2, _, err := env.Backend.Bugs().New("second bug title", "second bug body")
	require.NoError(t, err)

	opts := bugShowOptions{format: "default"}
	err = runBugShow(env, opts, []string{b1ID.Human(), b2.Id().Human()})
	require.EqualError(t, err, "multiple bug ids are only supported with --format=json")
	require.Empty(t, env.Out.String())
}

func TestBugShowMultipleIDsRejectFieldOutput(t *testing.T) {
	env, b1ID := testenv.NewTestEnvAndBug(t)
	b2, _, err := env.Backend.Bugs().New("second bug title", "second bug body")
	require.NoError(t, err)

	opts := bugShowOptions{format: "json", fields: "labels"}
	err = runBugShow(env, opts, []string{b1ID.Human(), b2.Id().Human()})
	require.EqualError(t, err, "multiple bug ids are not supported with --field")
	require.Empty(t, env.Out.String())
}

func TestBugShowJsonUnknownSecondIDPrintsNoPartialJSON(t *testing.T) {
	env, b1ID := testenv.NewTestEnvAndBug(t)

	opts := bugShowOptions{format: "json"}
	err := runBugShow(env, opts, []string{b1ID.Human(), "deadbee"})
	require.Error(t, err)
	require.Empty(t, env.Out.String())
}

func TestBugShowJsonMultipleIDsMatchSingleIDRepresentations(t *testing.T) {
	env, b1ID := testenv.NewTestEnvAndBug(t)
	b2, _, err := env.Backend.Bugs().New("second bug title", "second bug body")
	require.NoError(t, err)

	opts := bugShowOptions{format: "json"}
	require.NoError(t, runBugShow(env, opts, []string{b1ID.Human()}))
	var single map[string]any
	require.NoError(t, json.Unmarshal(env.Out.Bytes(), &single))
	env.Out.Reset()

	require.NoError(t, runBugShow(env, opts, []string{b1ID.Human(), b2.Id().Human()}))
	var batch []map[string]any
	require.NoError(t, json.Unmarshal(env.Out.Bytes(), &batch))
	require.Equal(t, single, batch[0])
}

func TestBugShowJsonThreeIDsPreserveArgumentOrder(t *testing.T) {
	env, b1ID := testenv.NewTestEnvAndBug(t)
	b2, _, err := env.Backend.Bugs().New("second bug title", "second bug body")
	require.NoError(t, err)
	b3, _, err := env.Backend.Bugs().New("third bug title", "third bug body")
	require.NoError(t, err)

	opts := bugShowOptions{format: "json"}
	require.NoError(t, runBugShow(env, opts, []string{
		b3.Id().Human(), b1ID.Human(), b2.Id().Human(),
	}))

	var got []map[string]any
	require.NoError(t, json.Unmarshal(env.Out.Bytes(), &got))
	require.Equal(t, []any{
		b3.Id().Human(), b1ID.Human(), b2.Id().Human(),
	}, []any{got[0]["human_id"], got[1]["human_id"], got[2]["human_id"]})
}

func TestBugShowJsonAmbiguousIDPrintsNoPartialJSON(t *testing.T) {
	env, b1ID := testenv.NewTestEnvAndBug(t)
	prefixes := map[string]struct{}{b1ID.Human()[:1]: {}}
	ambiguousPrefix := ""
	for i := 0; i < 256 && ambiguousPrefix == ""; i++ {
		b, _, err := env.Backend.Bugs().New(
			fmt.Sprintf("collision candidate %d", i),
			"collision candidate body",
		)
		require.NoError(t, err)

		prefix := b.Id().Human()[:1]
		if _, exists := prefixes[prefix]; exists {
			ambiguousPrefix = prefix
		} else {
			prefixes[prefix] = struct{}{}
		}
	}
	require.NotEmpty(t, ambiguousPrefix)

	opts := bugShowOptions{format: "json"}
	err := runBugShow(env, opts, []string{b1ID.Human(), ambiguousPrefix})
	require.Error(t, err)
	require.Empty(t, env.Out.String())
}

func TestBugShowJsonDuplicateIDsRemainDuplicated(t *testing.T) {
	env, bugID := testenv.NewTestEnvAndBug(t)

	opts := bugShowOptions{format: "json"}
	require.NoError(t, runBugShow(env, opts, []string{bugID.Human(), bugID.Human()}))

	var got []map[string]any
	require.NoError(t, json.Unmarshal(env.Out.Bytes(), &got))
	require.Len(t, got, 2)
	require.Equal(t, got[0], got[1])
}

func TestBugShowCommandAcceptsMultipleIDs(t *testing.T) {
	env, b1ID := testenv.NewTestEnvAndBug(t)
	b2, _, err := env.Backend.Bugs().New("second bug title", "second bug body")
	require.NoError(t, err)

	cmd := newBugShowCommand(env)
	cmd.PreRunE = nil
	cmd.SetArgs([]string{"--format=json", b1ID.Human(), b2.Id().Human()})
	require.NoError(t, cmd.Execute())

	var got []map[string]any
	require.NoError(t, json.Unmarshal(env.Out.Bytes(), &got))
	require.Len(t, got, 2)
}
