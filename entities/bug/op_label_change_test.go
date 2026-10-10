package bug

import (
	"testing"

	"github.com/git-bug/git-bug/entities/common"
	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/entity/dag/dagtest"
)

func TestLabelChangeSerialize(t *testing.T) {
	dagtest.SerializeRoundTripTest(t, operationUnmarshaler, func(author identity.Interface, unixTime int64) (*LabelChangeOperation, entity.Resolvers) {
		return NewLabelChangeOperation(author, unixTime, []common.Label{"added"}, []common.Label{"removed"}), nil
	})
	dagtest.SerializeRoundTripTest(t, operationUnmarshaler, func(author identity.Interface, unixTime int64) (*LabelChangeOperation, entity.Resolvers) {
		return NewLabelChangeOperation(author, unixTime, []common.Label{"added"}, nil), nil
	})
	dagtest.SerializeRoundTripTest(t, operationUnmarshaler, func(author identity.Interface, unixTime int64) (*LabelChangeOperation, entity.Resolvers) {
		return NewLabelChangeOperation(author, unixTime, nil, []common.Label{"removed"}), nil
	})
}
