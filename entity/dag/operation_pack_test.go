package dag

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/MichaelMure/git-bug/identity"
)

func TestOperationPackReadWrite(t *testing.T) {
	repo, id1, _, resolver, def := makeTestContext()

	blobHash1, err := repo.StoreData(randomData())
	require.NoError(t, err)

	blobHash2, err := repo.StoreData(randomData())
	require.NoError(t, err)

	opp := &operationPack{
		Author: id1,
		Operations: []Operation{
			newOp1(id1, "foo", blobHash1, blobHash2),
			newOp2(id1, "bar"),
		},
		CreateTime: 123,
		EditTime:   456,
	}

	commitHash, err := opp.Write(def, repo)
	require.NoError(t, err)

	commit, err := repo.ReadCommit(commitHash)
	require.NoError(t, err)

	opp2, err := readOperationPack(def, repo, resolver, commit)
	require.NoError(t, err)

	require.Equal(t, opp, opp2)

	// make sure we get the same Id with the same data
	opp3 := &operationPack{
		Author: id1,
		Operations: []Operation{
			newOp1(id1, "foo", blobHash1, blobHash2),
			newOp2(id1, "bar"),
		},
		CreateTime: 123,
		EditTime:   456,
	}
	require.Equal(t, opp.Id(), opp3.Id())
}

func TestOperationPackSignedReadWrite(t *testing.T) {
	repo, id1, _, resolver, def := makeTestContext()

	err := id1.(*identity.Identity).Mutate(repo, func(orig *identity.Mutator) {
		orig.Keys = append(orig.Keys, identity.GenerateKey())
	})
	require.NoError(t, err)

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

	opp2, err := readOperationPack(def, repo, resolver, commit)
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

func randomData() []byte {
	var letterRunes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	b := make([]byte, 32)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return b
}
