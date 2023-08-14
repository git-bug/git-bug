package bug

import (
	"testing"

	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/entity/dag"
)

func TestLabelChangeSerialize(t *testing.T) {
	dag.SerializeRoundTripTest(t, operationUnmarshaler, func(author entity.Identity, unixTime int64) (*LabelChangeOperation, entity.Resolvers) {
		return NewLabelChangeOperation(author, unixTime, []Label{"added"}, []Label{"removed"}), nil
	})
	dag.SerializeRoundTripTest(t, operationUnmarshaler, func(author entity.Identity, unixTime int64) (*LabelChangeOperation, entity.Resolvers) {
		return NewLabelChangeOperation(author, unixTime, []Label{"added"}, nil), nil
	})
	dag.SerializeRoundTripTest(t, operationUnmarshaler, func(author entity.Identity, unixTime int64) (*LabelChangeOperation, entity.Resolvers) {
		return NewLabelChangeOperation(author, unixTime, nil, []Label{"removed"}), nil
	})
}
