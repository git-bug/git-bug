package tests

import (
	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/bug/operations"
	"github.com/MichaelMure/git-bug/repository"
	"testing"
)

var (
	rene = bug.Person{
		Name:  "René Descartes",
		Email: "rene@descartes.fr",
	}

	createOp      = operations.NewCreateOp(rene, "title", "message", nil)
	setTitleOp    = operations.NewSetTitleOp(rene, "title2", "title1")
	addCommentOp  = operations.NewAddCommentOp(rene, "message2", nil)
	setStatusOp   = operations.NewSetStatusOp(rene, bug.ClosedStatus)
	labelChangeOp = operations.NewLabelChangeOperation(rene, []bug.Label{"added"}, []bug.Label{"removed"})
	mockRepo      = repository.NewMockRepoForTest()
)

func TestOpIterator(t *testing.T) {

	bug1 := bug.NewBug()

	bug1.Append(createOp)
	bug1.Append(setTitleOp)
	bug1.Append(addCommentOp)
	bug1.Append(setStatusOp)
	bug1.Append(labelChangeOp)
	bug1.Commit(mockRepo)

	bug1.Append(setTitleOp)
	bug1.Append(setTitleOp)
	bug1.Append(setTitleOp)
	bug1.Commit(mockRepo)

	bug1.Append(setTitleOp)
	bug1.Append(setTitleOp)
	bug1.Append(setTitleOp)

	it := bug.NewOperationIterator(bug1)

	counter := 0
	for it.Next() {
		_ = it.Value()
		counter++
	}

	if counter != 11 {
		t.Fatalf("Wrong count of value iterated (%d instead of 8)", counter)
	}
}
