package bug

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/MichaelMure/git-bug/identity"
	"github.com/MichaelMure/git-bug/repository"
)

func TestBugId(t *testing.T) {
	mockRepo := repository.NewMockRepoForTest()

	bug1 := NewBug()

	rene := identity.NewIdentity("René Descartes", "rene@descartes.fr")
	createOp := NewCreateOp(rene, time.Now().Unix(), "title", "message", nil)

	bug1.Append(createOp)

	err := bug1.Commit(mockRepo)

	if err != nil {
		t.Fatal(err)
	}

	bug1.Id()
}

func TestBugValidity(t *testing.T) {
	mockRepo := repository.NewMockRepoForTest()

	bug1 := NewBug()

	rene := identity.NewIdentity("René Descartes", "rene@descartes.fr")
	createOp := NewCreateOp(rene, time.Now().Unix(), "title", "message", nil)

	if bug1.Validate() == nil {
		t.Fatal("Empty bug should be invalid")
	}

	bug1.Append(createOp)

	if bug1.Validate() != nil {
		t.Fatal("Bug with just a CreateOp should be valid")
	}

	err := bug1.Commit(mockRepo)
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

func TestBugCommitLoadRemove(t *testing.T) {
	bug1 := NewBug()

	rene := identity.NewIdentity("René Descartes", "rene@descartes.fr")
	createOp := NewCreateOp(rene, time.Now().Unix(), "title", "message", nil)
	setTitleOp := NewSetTitleOp(rene, time.Now().Unix(), "title2", "title1")
	addCommentOp := NewAddCommentOp(rene, time.Now().Unix(), "message2", nil)

	bug1.Append(createOp)
	bug1.Append(setTitleOp)

	repo := repository.NewMockRepoForTest()

	assert.True(t, bug1.NeedCommit())

	err := bug1.Commit(repo)
	assert.Nil(t, err)
	assert.False(t, bug1.NeedCommit())

	bug2, err := ReadLocalBug(repo, bug1.Id())
	assert.NoError(t, err)
	equivalentBug(t, bug1, bug2)

	// add more op

	bug1.Append(addCommentOp)

	assert.True(t, bug1.NeedCommit())

	err = bug1.Commit(repo)
	assert.Nil(t, err)
	assert.False(t, bug1.NeedCommit())

	bug3, err := ReadLocalBug(repo, bug1.Id())
	assert.NoError(t, err)
	equivalentBug(t, bug1, bug3)

	err = RemoveLocalBug(repo, bug1.Id())
	assert.NoError(t, err)

	streamedBugs := ReadAllLocalBugs(repo)
	assert.Equal(t, 0, len(streamedBugs))
}

func equivalentBug(t *testing.T, expected, actual *Bug) {
	assert.Equal(t, len(expected.packs), len(actual.packs))

	for i := range expected.packs {
		for j := range expected.packs[i].Operations {
			actual.packs[i].Operations[j].base().id = expected.packs[i].Operations[j].base().id
		}
	}

	assert.Equal(t, expected, actual)
}
