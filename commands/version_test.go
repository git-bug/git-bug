package commands

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/commands/execenv"
)

func TestRawVersion(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "version alone",
			in:   "v1.2.3",
			want: "v1.2.3",
		},
		{
			name: "version and commit",
			in:   "v1.2.3 8a3f1b2c9d0e",
			want: "v1.2.3",
		},
		{
			name: "dirty build",
			in:   "v1.2.3 8a3f1b2c9d0e/dirty",
			want: "v1.2.3",
		},
		{
			name: "no build info",
			in:   "undefined (no build info)\n",
			want: "undefined",
		},
		{
			name: "full documented shape",
			in:   "v1.2.3 8a3f1b2c9d0e go1.21.0 linux amd64",
			want: "v1.2.3",
		},
		{
			name: "empty",
			in:   "",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, rawVersion(tt.in))
		})
	}
}

const testFullVersion = "v1.2.3 8a3f1b2c9d0e go1.21.0 linux amd64"

// runVersion executes `version` with the given args against a command whose
// root carries testFullVersion, and returns what was printed.
func runVersion(t *testing.T, args ...string) string {
	t.Helper()

	env := execenv.NewTestEnv(t)
	cmd := newVersionCommand(env)
	cmd.Version = testFullVersion
	cmd.SetArgs(args)
	cmd.SetOut(env.Out.Raw())
	cmd.SetErr(env.Out.Raw())
	require.NoError(t, cmd.Execute())

	return env.Out.Raw().(*bytes.Buffer).String()
}

// The whole point of --raw is that its output is embeddable: one line, no
// "git-bug" prefix, no build metadata.
func TestVersionCommandRaw(t *testing.T) {
	assert.Equal(t, "v1.2.3\n", runVersion(t, "--raw"))
}

// Without --raw the output is unchanged: the flag must not alter the default
// form that scripts and users already parse.
func TestVersionCommandDefaultUnchanged(t *testing.T) {
	assert.Equal(t, execenv.RootCommandName+" "+testFullVersion, runVersion(t))
}
