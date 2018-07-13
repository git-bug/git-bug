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

	if len(bug1.HumanId()) == 0 {
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
		t.Fatal("Bug with just a CREATE should be valid")
	}

	bug1.Append(createOp)

	if bug1.IsValid() {
		t.Fatal("Bug with multiple CREATE should be invalid")
	}

	bug1.Commit()

	if bug1.IsValid() {
		t.Fatal("Bug with multiple CREATE should be invalid")
	}
}
