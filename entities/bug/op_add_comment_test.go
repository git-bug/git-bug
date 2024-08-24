package bug

import (
	"testing"

	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/entity/dag"
	"github.com/git-bug/git-bug/repository"
)

func TestAddCommentSerialize(t *testing.T) {
	dag.SerializeRoundTripTest(t, operationUnmarshaler, func(author identity.Interface, unixTime int64) (*AddCommentOperation, entity.Resolvers) {
		return NewAddCommentOp(author, unixTime, "message", nil), nil
	})
	dag.SerializeRoundTripTest(t, operationUnmarshaler, func(author identity.Interface, unixTime int64) (*AddCommentOperation, entity.Resolvers) {
		return NewAddCommentOp(author, unixTime, "message", []repository.Hash{"hash1", "hash2"}), nil
	})
}
