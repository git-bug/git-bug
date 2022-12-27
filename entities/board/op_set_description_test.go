package board

import (
	"testing"

	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/entity/dag"
)

func TestSetDescriptionSerialize(t *testing.T) {
	dag.SerializeRoundTripTest(t, operationUnmarshaler, func(author identity.Interface, unixTime int64) (*SetDescriptionOperation, entity.Resolvers) {
		return NewSetDescriptionOp(author, unixTime, "description", "was"), nil
	})
}
