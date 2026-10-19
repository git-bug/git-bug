package board

import (
	"testing"

	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/entity/dag/dagtest"
)

func TestAddItemEntityOpSerialize(t *testing.T) {
	dagtest.SerializeRoundTripTest(t, operationUnmarshaler, func(author identity.Interface, unixTime int64) (*AddItemEntityOperation, entity.Resolvers) {
		return NewAddItemEntityOp(author, unixTime, entity.DeriveId([]byte("column")), EntityTypeBug, entity.DeriveId([]byte("bug"))), nil
	})
}
