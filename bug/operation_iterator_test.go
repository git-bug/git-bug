package bug

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/MichaelMure/git-bug/identity"
	"github.com/MichaelMure/git-bug/repository"
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

	rene := identity.NewIdentity("René Descartes", "rene@descartes.fr")
	err := rene.Commit(mockRepo)
	require.NoError(t, err)

	unix := time.Now().Unix()

	createOp := NewCreateOp(rene, unix, "title", "message", nil)
	addCommentOp := NewAddCommentOp(rene, unix, "message2", nil)
	setStatusOp := NewSetStatusOp(rene, unix, ClosedStatus)
	labelChangeOp := NewLabelChangeOperation(rene, unix, []Label{"added"}, []Label{"removed"})

	var i int
	genTitleOp := func() Operation {
		i++
		return NewSetTitleOp(rene, unix, fmt.Sprintf("title%d", i), "")
	}

	bug1 := NewBug()

	// first pack
	bug1.Append(createOp)
	bug1.Append(addCommentOp)
	bug1.Append(setStatusOp)
	bug1.Append(labelChangeOp)
	err = bug1.Commit(mockRepo)
	require.NoError(t, err)

	// second pack
	bug1.Append(genTitleOp())
	bug1.Append(genTitleOp())
	bug1.Append(genTitleOp())
	err = bug1.Commit(mockRepo)
	require.NoError(t, err)

	// staging
	bug1.Append(genTitleOp())
	bug1.Append(genTitleOp())
	bug1.Append(genTitleOp())

	it := NewOperationIterator(bug1)

	counter := 0
	for it.Next() {
		_ = it.Value()
		counter++
	}

	require.Equal(t, 10, counter)
}
