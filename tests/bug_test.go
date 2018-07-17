package tests

import (
	"github.com/MichaelMure/git-bug/bug"
	"testing"
)

func TestBugId(t *testing.T) {
	bug1, err := bug.NewBug()
	if err != nil {
		t.Error(err)
	}

	if len(bug1.Id()) == 0 {
		t.Fatal("Bug doesn't have a human readable identifier")
	}
}

func TestBugValidity(t *testing.T) {
	bug1, err := bug.NewBug()
	if err != nil {
		t.Error(err)
	}

	if bug1.IsValid() {
		t.Fatal("Empty bug should be invalid")
	}

	bug1.Append(createOp)

	if !bug1.IsValid() {
		t.Fatal("Bug with just a CreateOp should be valid")
	}

	bug1.Append(createOp)

	if bug1.IsValid() {
		t.Fatal("Bug with multiple CreateOp should be invalid")
	}

	bug1.Commit(mockRepo)

	if bug1.IsValid() {
		t.Fatal("Bug with multiple CreateOp should be invalid")
	}
}

//func TestBugSerialisation(t *testing.T) {
//	bug1, err := bug.NewBug()
//	if err != nil {
//		t.Error(err)
//	}
//
//	bug1.Append(createOp)
//	bug1.Append(setTitleOp)
//	bug1.Append(setTitleOp)
//	bug1.Append(addCommentOp)
//
//	repo := repository.NewMockRepoForTest()
//
//	bug1.Commit(repo)
//
//	bug2, err := bug.ReadBug(repo, bug.BugsRefPattern+bug1.Id())
//	if err != nil {
//		t.Error(err)
//	}
//
//	if !reflect.DeepEqual(bug1, bug2) {
//		t.Fatalf("%v different than %v", bug1, bug2)
//	}
//}
