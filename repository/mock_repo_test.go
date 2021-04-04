package repository

import (
	"testing"
)

func TestMockRepo(t *testing.T) {
	creator := func(bare bool) TestedRepo { return NewMockRepo() }
	cleaner := func(repos ...Repo) {}

	RepoTest(t, creator, cleaner)
}
