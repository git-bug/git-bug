package bug

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/entities/common"
	"github.com/git-bug/git-bug/entities/identity"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/entity/dag"
	"github.com/git-bug/git-bug/repository"
)

func TestCreate(t *testing.T) {
	repo := repository.NewMockRepo()

	rene, err := identity.NewIdentity(repo, "René Descartes", "rene@descartes.fr")
	require.NoError(t, err)

	b, op, err := Create(rene, time.Now().Unix(), "title", "message", nil, nil)
	require.NoError(t, err)

	require.Equal(t, "title", op.Title)
	require.Equal(t, "message", op.Message)

	// Create generate the initial operation and create a new timeline item
	snap := b.Snapshot()
	require.Equal(t, common.OpenStatus, snap.Status)
	require.Equal(t, rene, snap.Author)
	require.Equal(t, "title", snap.Title)
	require.Len(t, snap.Operations, 1)
	require.Equal(t, op, snap.Operations[0])

	require.Len(t, snap.Timeline, 1)
	require.Equal(t, entity.CombineIds(b.Id(), op.Id()), snap.Timeline[0].CombinedId())
	require.Equal(t, rene, snap.Timeline[0].(*CreateTimelineItem).Author)
	require.Equal(t, "message", snap.Timeline[0].(*CreateTimelineItem).Message)
}

func TestCreateSerialize(t *testing.T) {
	dag.SerializeRoundTripTest(t, operationUnmarshaler, func(author identity.Interface, unixTime int64) (*CreateOperation, entity.Resolvers) {
		return NewCreateOp(author, unixTime, "title", "message", nil), nil
	})
	dag.SerializeRoundTripTest(t, operationUnmarshaler, func(author identity.Interface, unixTime int64) (*CreateOperation, entity.Resolvers) {
		return NewCreateOp(author, unixTime, "title", "message", []repository.Hash{"hash1", "hash2"}), nil
	})
}
