package board

import (
	"testing"

	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/entity/dag"
	"github.com/git-bug/git-bug/repository"
)

func TestAddItemDraftOpSerialize(t *testing.T) {
	dag.SerializeRoundTripTest(t, operationUnmarshaler, func(author identity.Interface, unixTime int64) (*AddItemDraftOperation, entity.Resolvers) {
		return NewAddItemDraftOp(author, unixTime, "foo", "title", "message", nil), nil
	})
	dag.SerializeRoundTripTest(t, operationUnmarshaler, func(author identity.Interface, unixTime int64) (*AddItemDraftOperation, entity.Resolvers) {
		return NewAddItemDraftOp(author, unixTime, "foo", "title", "message", []repository.Hash{"hash1", "hash2"}), nil
	})
}
