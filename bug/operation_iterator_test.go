package bug

import (
	"github.com/MichaelMure/git-bug/identity"
	"github.com/MichaelMure/git-bug/repository"
	"testing"
	"time"
)

var (
	rene = identity.NewIdentity("René Descartes", "rene@descartes.fr")
	unix = time.Now().Unix()

	createOp      = NewCreateOp(rene, unix, "title", "message", nil)
	setTitleOp    = NewSetTitleOp(rene, unix, "title2", "title1")
	addCommentOp  = NewAddCommentOp(rene, unix, "message2", nil)
	setStatusOp   = NewSetStatusOp(rene, unix, ClosedStatus)
	labelChangeOp = NewLabelChangeOperation(rene, unix, []Label{"added"}, []Label{"removed"})
)

func TestOpIterator(t *testing.T) {
	mockRepo := repository.NewMockRepoForTest()

	bug1 := NewBug()

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

	it := NewOperationIterator(bug1)

	counter := 0
	for it.Next() {
		_ = it.Value()
		counter++
	}

	if counter != 11 {
		t.Fatalf("Wrong count of value iterated (%d instead of 8)", counter)
	}
}
