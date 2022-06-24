package repository

import (
	"testing"
)

func TestMockRepo(t *testing.T) {
	creator := func(t testing.TB, bare bool) TestedRepo { return NewMockRepo() }

	RepoTest(t, creator)
}
