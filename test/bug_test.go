package test

import (
	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/bug/operations"
	"testing"
)

func TestBug(t *testing.T) {
	var rene = bug.Person{
		Name:  "René Descartes",
		Email: "rene@descartes.fr",
	}

	var createOp = operations.NewCreateOp(rene, "title", "message")

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
