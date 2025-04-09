package board

import (
	"testing"

	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/entity/dag"
)

func TestSetTitleSerialize(t *testing.T) {
	dag.SerializeRoundTripTest(t, operationUnmarshaler, func(author identity.Interface, unixTime int64) (*SetTitleOperation, entity.Resolvers) {
		return NewSetTitleOp(author, unixTime, "title", "was"), nil
	})
}
