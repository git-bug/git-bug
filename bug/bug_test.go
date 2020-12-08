package bug

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/MichaelMure/git-bug/identity"
	"github.com/MichaelMure/git-bug/repository"
)

func TestBugId(t *testing.T) {
	mockRepo := repository.NewMockRepoForTest()

	bug1 := NewBug()

	rene := identity.NewIdentity("René Descartes", "rene@descartes.fr")
	err := rene.Commit(mockRepo)
	require.NoError(t, err)

	createOp := NewCreateOp(rene, time.Now().Unix(), "title", "message", nil)

	bug1.Append(createOp)

	err = bug1.Commit(mockRepo)

	if err != nil {
		t.Fatal(err)
	}

	bug1.Id()
}

func TestBugValidity(t *testing.T) {
	mockRepo := repository.NewMockRepoForTest()

	bug1 := NewBug()

	rene := identity.NewIdentity("René Descartes", "rene@descartes.fr")
	err := rene.Commit(mockRepo)
	require.NoError(t, err)

	createOp := NewCreateOp(rene, time.Now().Unix(), "title", "message", nil)

	if bug1.Validate() == nil {
		t.Fatal("Empty bug should be invalid")
	}

	bug1.Append(createOp)

	if bug1.Validate() != nil {
		t.Fatal("Bug with just a CreateOp should be valid")
	}

	err = bug1.Commit(mockRepo)
	if err != nil {
		t.Fatal(err)
	}

	bug1.Append(createOp)

	if bug1.Validate() == nil {
		t.Fatal("Bug with multiple CreateOp should be invalid")
	}

	err = bug1.Commit(mockRepo)
	if err == nil {
		t.Fatal("Invalid bug should not commit")
	}
}

func TestBugCommitLoad(t *testing.T) {
	repo := repository.NewMockRepoForTest()

	bug1 := NewBug()

	rene := identity.NewIdentity("René Descartes", "rene@descartes.fr")
	err := rene.Commit(repo)
	require.NoError(t, err)

	createOp := NewCreateOp(rene, time.Now().Unix(), "title", "message", nil)
	setTitleOp := NewSetTitleOp(rene, time.Now().Unix(), "title2", "title1")
	addCommentOp := NewAddCommentOp(rene, time.Now().Unix(), "message2", nil)

	bug1.Append(createOp)
	bug1.Append(setTitleOp)

	require.True(t, bug1.NeedCommit())

	err = bug1.Commit(repo)
	require.Nil(t, err)
	require.False(t, bug1.NeedCommit())

	bug2, err := ReadLocal(repo, bug1.Id())
	require.NoError(t, err)
	equivalentBug(t, bug1, bug2)

	// add more op

	bug1.Append(addCommentOp)

	require.True(t, bug1.NeedCommit())

	err = bug1.Commit(repo)
	require.Nil(t, err)
	require.False(t, bug1.NeedCommit())

	bug3, err := ReadLocal(repo, bug1.Id())
	require.NoError(t, err)
	equivalentBug(t, bug1, bug3)
}

func equivalentBug(t *testing.T, expected, actual *Bug) {
	require.Equal(t, len(expected.packs), len(actual.packs))

	for i := range expected.packs {
		for j := range expected.packs[i].Operations {
			actual.packs[i].Operations[j].base().id = expected.packs[i].Operations[j].base().id
		}
	}

	require.Equal(t, expected, actual)
}

func TestBugRemove(t *testing.T) {
	repo := repository.CreateGoGitTestRepo(false)
	remoteA := repository.CreateGoGitTestRepo(true)
	remoteB := repository.CreateGoGitTestRepo(true)
	defer repository.CleanupTestRepos(repo, remoteA, remoteB)

	err := repo.AddRemote("remoteA", remoteA.GetLocalRemote())
	require.NoError(t, err)

	err = repo.AddRemote("remoteB", remoteB.GetLocalRemote())
	require.NoError(t, err)

	// generate a bunch of bugs
	rene := identity.NewIdentity("René Descartes", "rene@descartes.fr")
	err = rene.Commit(repo)
	require.NoError(t, err)

	for i := 0; i < 100; i++ {
		b := NewBug()
		createOp := NewCreateOp(rene, time.Now().Unix(), "title", fmt.Sprintf("message%v", i), nil)
		b.Append(createOp)
		err = b.Commit(repo)
		require.NoError(t, err)
	}

	// and one more for testing
	b := NewBug()
	createOp := NewCreateOp(rene, time.Now().Unix(), "title", "message", nil)
	b.Append(createOp)
	err = b.Commit(repo)
	require.NoError(t, err)

	_, err = Push(repo, "remoteA")
	require.NoError(t, err)

	_, err = Push(repo, "remoteB")
	require.NoError(t, err)

	_, err = Fetch(repo, "remoteA")
	require.NoError(t, err)

	_, err = Fetch(repo, "remoteB")
	require.NoError(t, err)

	err = RemoveBug(repo, b.Id())
	require.NoError(t, err)

	_, err = ReadLocal(repo, b.Id())
	require.Error(t, ErrBugNotExist, err)

	_, err = ReadRemote(repo, "remoteA", b.Id())
	require.Error(t, ErrBugNotExist, err)

	_, err = ReadRemote(repo, "remoteB", b.Id())
	require.Error(t, ErrBugNotExist, err)

	ids, err := ListLocalIds(repo)
	require.NoError(t, err)
	require.Len(t, ids, 100)
}
