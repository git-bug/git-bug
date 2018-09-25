package tests

import (
	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/operations"
	"github.com/MichaelMure/git-bug/repository"
	"testing"
	"time"
)

var (
	rene = bug.Person{
		Name:  "René Descartes",
		Email: "rene@descartes.fr",
	}

	unix = time.Now().Unix()

	createOp      = operations.NewCreateOp(rene, unix, "title", "message", nil)
	setTitleOp    = operations.NewSetTitleOp(rene, unix, "title2", "title1")
	addCommentOp  = operations.NewAddCommentOp(rene, unix, "message2", nil)
	setStatusOp   = operations.NewSetStatusOp(rene, unix, bug.ClosedStatus)
	labelChangeOp = operations.NewLabelChangeOperation(rene, unix, []bug.Label{"added"}, []bug.Label{"removed"})
)

func TestOpIterator(t *testing.T) {
	mockRepo := repository.NewMockRepoForTest()

	bug1 := bug.NewBug()

	// first pack
	bug1.Append(createOp)
	bug1.Append(setTitleOp)
	bug1.Append(addCommentOp)
	bug1.Append(setStatusOp)
	bug1.Append(labelChangeOp)
	bug1.Commit(mockRepo)

	// second pack
	bug1.Append(setTitleOp)
	bug1.Append(setTitleOp)
	bug1.Append(setTitleOp)
	bug1.Commit(mockRepo)

	// staging
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
