package bug

import (
	"testing"
	"time"

	"github.com/MichaelMure/git-bug/identity"
	"github.com/MichaelMure/git-bug/repository"
	"github.com/stretchr/testify/assert"
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

func TestBugCommitLoad(t *testing.T) {
	bug1 := NewBug()

	rene := identity.NewIdentity("René Descartes", "rene@descartes.fr")
	createOp := NewCreateOp(rene, time.Now().Unix(), "title", "message", nil)
	setTitleOp := NewSetTitleOp(rene, time.Now().Unix(), "title2", "title1")
	addCommentOp := NewAddCommentOp(rene, time.Now().Unix(), "message2", nil)

	bug1.Append(createOp)
	bug1.Append(setTitleOp)
	bug1.Append(setTitleOp)
	bug1.Append(addCommentOp)

	repo := repository.NewMockRepoForTest()

	err := bug1.Commit(repo)
	assert.Nil(t, err)

	bug2, err := ReadLocalBug(repo, bug1.Id())
	assert.NoError(t, err)
	equivalentBug(t, bug1, bug2)

	// add more op

	bug1.Append(setTitleOp)
	bug1.Append(addCommentOp)

	err = bug1.Commit(repo)
	assert.Nil(t, err)

	bug3, err := ReadLocalBug(repo, bug1.Id())
	assert.NoError(t, err)
	equivalentBug(t, bug1, bug3)
}

func equivalentBug(t *testing.T, expected, actual *Bug) {
	assert.Equal(t, len(expected.packs), len(actual.packs))

	for i := range expected.packs {
		for j := range expected.packs[i].Operations {
			actual.packs[i].Operations[j].base().hash = expected.packs[i].Operations[j].base().hash
		}
	}

	assert.Equal(t, expected, actual)
}
