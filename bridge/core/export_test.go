package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/entities/bug"
	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity/dag"
	"github.com/git-bug/git-bug/repository"
)

func TestSkipReasonNoTokenActor(t *testing.T) {
	author, err := identity.NewIdentity(repository.NewMockRepo(), "Jane", "jane@example.com")
	require.NoError(t, err)

	synced := bug.NewSetTitleOp(author, 1, "synced", "")
	synced.SetMetadata("remote-id", "1")
	local := bug.NewSetTitleOp(author, 2, "local", "synced")
	noop := dag.NewNoOpOp[*bug.Snapshot](bug.NoOpOp, author, 3)

	assert.Equal(t, ReasonNothingExported,
		SkipReasonNoTokenActor([]dag.Operation{synced, noop}, "remote-id"),
		"an issue fully in sync is up to date, not skipped")
	assert.Equal(t, ReasonNoTokenActor,
		SkipReasonNoTokenActor([]dag.Operation{synced, local}, "remote-id"),
		"a local change nobody can push is a real skip")
}
