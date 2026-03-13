package bug

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/entity/dag"
	"github.com/git-bug/git-bug/repository"
)

func TestSetAssigneeSerialize(t *testing.T) {
	dag.SerializeRoundTripTest(t, operationUnmarshaler, func(author identity.Interface, unixTime int64) (*SetAssigneeOperation, entity.Resolvers) {
		return NewSetAssigneeOp(author, unixTime, "", nil), nil
	})
}

func TestSetAssigneeOperation(t *testing.T) {
	repo := repository.NewMockRepo()

	rene, err := identity.NewIdentity(repo, "René Descartes", "rene@descartes.fr")
	require.NoError(t, err)

	assignee, err := identity.NewIdentity(repo, "Isaac Newton", "isaac@newton.uk")
	require.NoError(t, err)

	unix := time.Now().Unix()

	b, _, err := Create(rene, unix, "title", "message", nil, nil)
	require.NoError(t, err)

	snap := b.Compile()
	require.Nil(t, snap.Assignee)

	_, err = SetAssignee(b, rene, unix, assignee.Id(), assignee, nil)
	require.NoError(t, err)

	snap = b.Compile()
	require.Equal(t, assignee.Id(), snap.Assignee.Id())

	_, err = Unassign(b, rene, unix, nil)
	require.NoError(t, err)

	snap = b.Compile()
	require.Nil(t, snap.Assignee)
}
