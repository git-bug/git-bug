package core

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/repository"
)

func TestLockBridges(t *testing.T) {
	repo := repository.NewMockRepo()

	unlock, err := lockBridges(repo)
	require.NoError(t, err)

	_, err = lockBridges(repo)
	require.ErrorIs(t, err, ErrBridgeRunning)

	unlock()

	unlock, err = lockBridges(repo)
	require.NoError(t, err)
	unlock()
}
