package tests

import (
	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/bug/operations"
	"testing"
)

var (
	rene = bug.Person{
		Name:  "René Descartes",
		Email: "rene@descartes.fr",
	}

	createOp   = operations.NewCreateOp(rene, "title", "message")
	setTitleOp = operations.NewSetTitleOp("title2")
)

func TestOpIterator(t *testing.T) {

	bug1, err := bug.NewBug()

	if err != nil {
		t.Fatal(err)
	}

	bug1.Append(createOp)
	bug1.Append(setTitleOp)
	bug1.Commit()

	bug1.Append(setTitleOp)
	bug1.Append(setTitleOp)
	bug1.Append(setTitleOp)
	bug1.Commit()

	bug1.Append(setTitleOp)
	bug1.Append(setTitleOp)
	bug1.Append(setTitleOp)

	it := bug.NewOperationIterator(bug1)

	counter := 0
	for it.Next() {
		_ = it.Value()
		counter++
	}

	if counter != 8 {
		t.Fatalf("Wrong count of value iterated (%d instead of 8)", counter)
	}
}
