package bug

import (
	"testing"

	"github.com/git-bug/git-bug/entities/common"
	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/entity/dag"
)

func TestSetStatusSerialize(t *testing.T) {
	dag.SerializeRoundTripTest(t, operationUnmarshaler, func(author identity.Interface, unixTime int64) (*SetStatusOperation, entity.Resolvers) {
		return NewSetStatusOp(author, unixTime, common.ClosedStatus), nil
	})
}
