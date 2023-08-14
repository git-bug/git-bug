package bug

import (
	"testing"

	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/entity/dag"
	"github.com/MichaelMure/git-bug/repository"
)

func TestAddCommentSerialize(t *testing.T) {
	dag.SerializeRoundTripTest(t, operationUnmarshaler, func(author entity.Identity, unixTime int64) (*AddCommentOperation, entity.Resolvers) {
		return NewAddCommentOp(author, unixTime, "message", nil), nil
	})
	dag.SerializeRoundTripTest(t, operationUnmarshaler, func(author entity.Identity, unixTime int64) (*AddCommentOperation, entity.Resolvers) {
		return NewAddCommentOp(author, unixTime, "message", []repository.Hash{"hash1", "hash2"}), nil
	})
}
