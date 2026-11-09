package config

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/entity/dag/dagtest"
)

func TestRepo(t *testing.T) {
	clocks, rene, isaac := makeTestContext(t)
	unix := time.Now().Unix()

	t.Run("validate", func(t *testing.T) {
		// the length is counted in characters: "é" is two bytes
		for _, description := range []string{"description", "", strings.Repeat("é", 256)} {
			op, err := NewRepoSetDescriptionOp(clocks, rene, unix, description)
			require.NoError(t, err)
			require.NoError(t, op.Validate())
		}

		for _, description := range []string{"multi\nline description", "description\u001b", strings.Repeat("a", 257)} {
			op, err := NewRepoSetDescriptionOp(clocks, rene, unix, description)
			require.NoError(t, err)
			require.Error(t, op.Validate())
		}
	})

	t.Run("serialize", func(t *testing.T) {
		for _, description := range []string{"description", ""} {
			dagtest.SerializeRoundTripTest(t, operationUnmarshaler, func(author identity.Interface, unixTime int64) (*RepoSetDescriptionOperation, entity.Resolvers) {
				op, err := NewRepoSetDescriptionOp(clocks, author, unixTime, description)
				require.NoError(t, err)
				return op, nil
			})
		}
	})

	t.Run("last writer wins", func(t *testing.T) {
		c, _, err := Create(clocks, rene, unix)
		require.NoError(t, err)
		_, err = RepoSetDescription(c, clocks, rene, unix, "first")
		require.NoError(t, err)
		_, err = RepoSetDescription(c, clocks, isaac, unix, "second")
		require.NoError(t, err)

		require.Equal(t, "second", c.Compile().Repo.Description)
	})
}
