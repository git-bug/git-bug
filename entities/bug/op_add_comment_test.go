package bug

import (
	"strings"
	"testing"

	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/entity/dag/dagtest"
	"github.com/git-bug/git-bug/repository"
)

func TestAddCommentSerialize(t *testing.T) {
	dagtest.SerializeRoundTripTest(t, operationUnmarshaler, func(author identity.Interface, unixTime int64) (*AddCommentOperation, entity.Resolvers) {
		return NewAddCommentOp(author, unixTime, "message", nil), nil
	})
	dagtest.SerializeRoundTripTest(t, operationUnmarshaler, func(author identity.Interface, unixTime int64) (*AddCommentOperation, entity.Resolvers) {
		return NewAddCommentOp(author, unixTime, "message",
			[]repository.Hash{
				repository.Hash(strings.Repeat("a", 40)),
				repository.Hash(strings.Repeat("b", 40)),
			}), nil
	})
}
