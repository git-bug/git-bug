package bug

import (
	"github.com/MichaelMure/git-bug/repository"
	"github.com/go-test/deep"
	"github.com/stretchr/testify/assert"

	"testing"
)

func TestBugId(t *testing.T) {
	mockRepo := repository.NewMockRepoForTest()

	bug1 := NewBug()

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

func TestBugSerialisation(t *testing.T) {
	bug1 := NewBug()

	bug1.Append(createOp)
	bug1.Append(setTitleOp)
	bug1.Append(setTitleOp)
	bug1.Append(addCommentOp)

	repo := repository.NewMockRepoForTest()

	err := bug1.Commit(repo)
	assert.Nil(t, err)

	bug2, err := ReadLocalBug(repo, bug1.Id())
	if err != nil {
		t.Error(err)
	}

	// ignore some fields
	bug2.packs[0].commitHash = bug1.packs[0].commitHash
	for i := range bug1.packs[0].Operations {
		bug2.packs[0].Operations[i].base().hash = bug1.packs[0].Operations[i].base().hash
	}

	// check hashes
	for i := range bug1.packs[0].Operations {
		if !bug2.packs[0].Operations[i].base().hash.IsValid() {
			t.Fatal("invalid hash")
		}
	}

	deep.CompareUnexportedFields = true
	if diff := deep.Equal(bug1, bug2); diff != nil {
		t.Fatal(diff)
	}
}
