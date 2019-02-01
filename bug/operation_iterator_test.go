package bug

import (
	"github.com/MichaelMure/git-bug/identity"
	"github.com/MichaelMure/git-bug/repository"
	"github.com/stretchr/testify/assert"

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

func ExampleOperationIterator() {
	b := NewBug()

	// add operations

	it := NewOperationIterator(b)

	for it.Next() {
		// do something with each operations
		_ = it.Value()
	}
}

func TestOpIterator(t *testing.T) {
	mockRepo := repository.NewMockRepoForTest()

	bug1 := NewBug()

	// first pack
	bug1.Append(createOp)
	bug1.Append(setTitleOp)
	bug1.Append(addCommentOp)
	bug1.Append(setStatusOp)
	bug1.Append(labelChangeOp)
	err := bug1.Commit(mockRepo)
	assert.NoError(t, err)

	// second pack
	bug1.Append(setTitleOp)
	bug1.Append(setTitleOp)
	bug1.Append(setTitleOp)
	err = bug1.Commit(mockRepo)
	assert.NoError(t, err)

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
