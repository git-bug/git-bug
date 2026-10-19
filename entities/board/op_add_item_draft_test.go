package board

import (
	"testing"

	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/entity/dag/dagtest"
)

func TestAddItemDraftOpSerialize(t *testing.T) {
	dagtest.SerializeRoundTripTest(t, operationUnmarshaler, func(author identity.Interface, unixTime int64) (*AddItemDraftOperation, entity.Resolvers) {
		return NewAddItemDraftOp(author, unixTime, entity.DeriveId([]byte("column")), "title"), nil
	})
}
