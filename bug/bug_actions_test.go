package bug

import (
	"github.com/MichaelMure/git-bug/util/test"
	"github.com/stretchr/testify/assert"

	"testing"
)

func TestPushPull(t *testing.T) {
	repoA, repoB, remote := test.SetupReposAndRemote(t)
	defer test.CleanupRepos(repoA, repoB, remote)

	err := rene.Commit(repoA)
	assert.NoError(t, err)

	bug1, _, err := Create(rene, unix, "bug1", "message")
	assert.NoError(t, err)
	err = bug1.Commit(repoA)
	assert.NoError(t, err)

	// A --> remote --> B
	_, err = Push(repoA, "origin")
	assert.NoError(t, err)

	err = Pull(repoB, "origin")
	assert.NoError(t, err)

	bugs := allBugs(t, ReadAllLocalBugs(repoB))

	if len(bugs) != 1 {
		t.Fatal("Unexpected number of bugs")
	}

	// B --> remote --> A
	bug2, _, err := Create(rene, unix, "bug2", "message")
	assert.NoError(t, err)
	err = bug2.Commit(repoB)
	assert.NoError(t, err)

	_, err = Push(repoB, "origin")
	assert.NoError(t, err)

	err = Pull(repoA, "origin")
	assert.NoError(t, err)

	bugs = allBugs(t, ReadAllLocalBugs(repoA))

	if len(bugs) != 2 {
		t.Fatal("Unexpected number of bugs")
	}
}

func allBugs(t testing.TB, bugs <-chan StreamedBug) []*Bug {
	var result []*Bug
	for streamed := range bugs {
		if streamed.Err != nil {
			t.Fatal(streamed.Err)
		}
		result = append(result, streamed.Bug)
	}
	return result
}

func TestRebaseTheirs(t *testing.T) {
	_RebaseTheirs(t)
}

func BenchmarkRebaseTheirs(b *testing.B) {
	for n := 0; n < b.N; n++ {
		_RebaseTheirs(b)
	}
}

func _RebaseTheirs(t testing.TB) {
	repoA, repoB, remote := test.SetupReposAndRemote(t)
	defer test.CleanupRepos(repoA, repoB, remote)

	bug1, _, err := Create(rene, unix, "bug1", "message")
	assert.NoError(t, err)
	err = bug1.Commit(repoA)
	assert.NoError(t, err)

	// A --> remote
	_, err = Push(repoA, "origin")
	assert.NoError(t, err)

	// remote --> B
	err = Pull(repoB, "origin")
	assert.NoError(t, err)

	bug2, err := ReadLocalBug(repoB, bug1.Id())
	assert.NoError(t, err)

	_, err = AddComment(bug2, rene, unix, "message2")
	assert.NoError(t, err)
	_, err = AddComment(bug2, rene, unix, "message3")
	assert.NoError(t, err)
	_, err = AddComment(bug2, rene, unix, "message4")
	assert.NoError(t, err)
	err = bug2.Commit(repoB)
	assert.NoError(t, err)

	// B --> remote
	_, err = Push(repoB, "origin")
	assert.NoError(t, err)

	// remote --> A
	err = Pull(repoA, "origin")
	assert.NoError(t, err)

	bugs := allBugs(t, ReadAllLocalBugs(repoB))

	if len(bugs) != 1 {
		t.Fatal("Unexpected number of bugs")
	}

	bug3, err := ReadLocalBug(repoA, bug1.Id())
	assert.NoError(t, err)

	if nbOps(bug3) != 4 {
		t.Fatal("Unexpected number of operations")
	}
}

func TestRebaseOurs(t *testing.T) {
	_RebaseOurs(t)
}

func BenchmarkRebaseOurs(b *testing.B) {
	for n := 0; n < b.N; n++ {
		_RebaseOurs(b)
	}
}

func _RebaseOurs(t testing.TB) {
	repoA, repoB, remote := test.SetupReposAndRemote(t)
	defer test.CleanupRepos(repoA, repoB, remote)

	bug1, _, err := Create(rene, unix, "bug1", "message")
	assert.NoError(t, err)
	err = bug1.Commit(repoA)
	assert.NoError(t, err)

	// A --> remote
	_, err = Push(repoA, "origin")
	assert.NoError(t, err)

	// remote --> B
	err = Pull(repoB, "origin")
	assert.NoError(t, err)

	_, err = AddComment(bug1, rene, unix, "message2")
	assert.NoError(t, err)
	_, err = AddComment(bug1, rene, unix, "message3")
	assert.NoError(t, err)
	_, err = AddComment(bug1, rene, unix, "message4")
	assert.NoError(t, err)
	err = bug1.Commit(repoA)
	assert.NoError(t, err)

	_, err = AddComment(bug1, rene, unix, "message5")
	assert.NoError(t, err)
	_, err = AddComment(bug1, rene, unix, "message6")
	assert.NoError(t, err)
	_, err = AddComment(bug1, rene, unix, "message7")
	assert.NoError(t, err)
	err = bug1.Commit(repoA)
	assert.NoError(t, err)

	_, err = AddComment(bug1, rene, unix, "message8")
	assert.NoError(t, err)
	_, err = AddComment(bug1, rene, unix, "message9")
	assert.NoError(t, err)
	_, err = AddComment(bug1, rene, unix, "message10")
	assert.NoError(t, err)
	err = bug1.Commit(repoA)
	assert.NoError(t, err)

	// remote --> A
	err = Pull(repoA, "origin")
	assert.NoError(t, err)

	bugs := allBugs(t, ReadAllLocalBugs(repoA))

	if len(bugs) != 1 {
		t.Fatal("Unexpected number of bugs")
	}

	bug2, err := ReadLocalBug(repoA, bug1.Id())
	assert.NoError(t, err)

	if nbOps(bug2) != 10 {
		t.Fatal("Unexpected number of operations")
	}
}

func nbOps(b *Bug) int {
	it := NewOperationIterator(b)
	counter := 0
	for it.Next() {
		counter++
	}
	return counter
}

func TestRebaseConflict(t *testing.T) {
	_RebaseConflict(t)
}

func BenchmarkRebaseConflict(b *testing.B) {
	for n := 0; n < b.N; n++ {
		_RebaseConflict(b)
	}
}

func _RebaseConflict(t testing.TB) {
	repoA, repoB, remote := test.SetupReposAndRemote(t)
	defer test.CleanupRepos(repoA, repoB, remote)

	bug1, _, err := Create(rene, unix, "bug1", "message")
	assert.NoError(t, err)
	err = bug1.Commit(repoA)
	assert.NoError(t, err)

	// A --> remote
	_, err = Push(repoA, "origin")
	assert.NoError(t, err)

	// remote --> B
	err = Pull(repoB, "origin")
	assert.NoError(t, err)

	_, err = AddComment(bug1, rene, unix, "message2")
	assert.NoError(t, err)
	_, err = AddComment(bug1, rene, unix, "message3")
	assert.NoError(t, err)
	_, err = AddComment(bug1, rene, unix, "message4")
	assert.NoError(t, err)
	err = bug1.Commit(repoA)
	assert.NoError(t, err)

	_, err = AddComment(bug1, rene, unix, "message5")
	assert.NoError(t, err)
	_, err = AddComment(bug1, rene, unix, "message6")
	assert.NoError(t, err)
	_, err = AddComment(bug1, rene, unix, "message7")
	assert.NoError(t, err)
	err = bug1.Commit(repoA)
	assert.NoError(t, err)

	_, err = AddComment(bug1, rene, unix, "message8")
	assert.NoError(t, err)
	_, err = AddComment(bug1, rene, unix, "message9")
	assert.NoError(t, err)
	_, err = AddComment(bug1, rene, unix, "message10")
	assert.NoError(t, err)
	err = bug1.Commit(repoA)
	assert.NoError(t, err)

	bug2, err := ReadLocalBug(repoB, bug1.Id())
	assert.NoError(t, err)

	_, err = AddComment(bug2, rene, unix, "message11")
	assert.NoError(t, err)
	_, err = AddComment(bug2, rene, unix, "message12")
	assert.NoError(t, err)
	_, err = AddComment(bug2, rene, unix, "message13")
	assert.NoError(t, err)
	err = bug2.Commit(repoB)
	assert.NoError(t, err)

	_, err = AddComment(bug2, rene, unix, "message14")
	assert.NoError(t, err)
	_, err = AddComment(bug2, rene, unix, "message15")
	assert.NoError(t, err)
	_, err = AddComment(bug2, rene, unix, "message16")
	assert.NoError(t, err)
	err = bug2.Commit(repoB)
	assert.NoError(t, err)

	_, err = AddComment(bug2, rene, unix, "message17")
	assert.NoError(t, err)
	_, err = AddComment(bug2, rene, unix, "message18")
	assert.NoError(t, err)
	_, err = AddComment(bug2, rene, unix, "message19")
	assert.NoError(t, err)
	err = bug2.Commit(repoB)
	assert.NoError(t, err)

	// A --> remote
	_, err = Push(repoA, "origin")
	assert.NoError(t, err)

	// remote --> B
	err = Pull(repoB, "origin")
	assert.NoError(t, err)

	bugs := allBugs(t, ReadAllLocalBugs(repoB))

	if len(bugs) != 1 {
		t.Fatal("Unexpected number of bugs")
	}

	bug3, err := ReadLocalBug(repoB, bug1.Id())
	assert.NoError(t, err)

	if nbOps(bug3) != 19 {
		t.Fatal("Unexpected number of operations")
	}

	// B --> remote
	_, err = Push(repoB, "origin")
	assert.NoError(t, err)

	// remote --> A
	err = Pull(repoA, "origin")
	assert.NoError(t, err)

	bugs = allBugs(t, ReadAllLocalBugs(repoA))

	if len(bugs) != 1 {
		t.Fatal("Unexpected number of bugs")
	}

	bug4, err := ReadLocalBug(repoA, bug1.Id())
	assert.NoError(t, err)

	if nbOps(bug4) != 19 {
		t.Fatal("Unexpected number of operations")
	}
}
