package repository

import (
	"testing"
)

func TestMockRepo(t *testing.T) {
	creator := func(t TestingT, bare bool) TestedRepo { return NewMockRepo() }

	RepoTest(t, creator)
}
