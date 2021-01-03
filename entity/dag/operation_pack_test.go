package dag

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOperationPackReadWrite(t *testing.T) {
	repo, id1, _, def := makeTestContext()

	opp := &operationPack{
		Author: id1,
		Operations: []Operation{
			newOp1(id1, "foo"),
			newOp2(id1, "bar"),
		},
		CreateTime: 123,
		EditTime:   456,
	}

	commitHash, err := opp.Write(def, repo)
	require.NoError(t, err)

	commit, err := repo.ReadCommit(commitHash)
	require.NoError(t, err)

	opp2, err := readOperationPack(def, repo, commit)
	require.NoError(t, err)

	require.Equal(t, opp, opp2)

	// make sure we get the same Id with the same data
	opp3 := &operationPack{
		Author: id1,
		Operations: []Operation{
			newOp1(id1, "foo"),
			newOp2(id1, "bar"),
		},
		CreateTime: 123,
		EditTime:   456,
	}
	require.Equal(t, opp.Id(), opp3.Id())
}
