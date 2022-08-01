package bug

import (
	"testing"

	"github.com/MichaelMure/git-bug/entity/dag"
	"github.com/MichaelMure/git-bug/identity"
	"github.com/MichaelMure/git-bug/repository"
)

func TestAddCommentSerialize(t *testing.T) {
	dag.SerializeRoundTripTest(t, func(author identity.Interface, unixTime int64) *AddCommentOperation {
		return NewAddCommentOp(author, unixTime, "message", nil)
	})
	dag.SerializeRoundTripTest(t, func(author identity.Interface, unixTime int64) *AddCommentOperation {
		return NewAddCommentOp(author, unixTime, "message", []repository.Hash{"hash1", "hash2"})
	})
}
