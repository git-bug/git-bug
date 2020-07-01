package bug

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MichaelMure/git-bug/identity"
	"github.com/MichaelMure/git-bug/repository"
)

func TestOperationPackSerialize(t *testing.T) {
	opp := &OperationPack{}

	repo := repository.NewMockRepoForTest()
	rene := identity.NewIdentity("René Descartes", "rene@descartes.fr")
	err := rene.Commit(repo)
	require.NoError(t, err)

	createOp := NewCreateOp(rene, time.Now().Unix(), "title", "message", nil)
	setTitleOp := NewSetTitleOp(rene, time.Now().Unix(), "title2", "title1")
	addCommentOp := NewAddCommentOp(rene, time.Now().Unix(), "message2", nil)
	setStatusOp := NewSetStatusOp(rene, time.Now().Unix(), ClosedStatus)
	labelChangeOp := NewLabelChangeOperation(rene, time.Now().Unix(), []Label{"added"}, []Label{"removed"})

	opp.Append(createOp)
	opp.Append(setTitleOp)
	opp.Append(addCommentOp)
	opp.Append(setStatusOp)
	opp.Append(labelChangeOp)

	opMeta := NewSetTitleOp(rene, time.Now().Unix(), "title3", "title2")
	opMeta.SetMetadata("key", "value")
	opp.Append(opMeta)

	assert.Equal(t, 1, len(opMeta.Metadata))

	opFile := NewAddCommentOp(rene, time.Now().Unix(), "message", []repository.Hash{
		"abcdef",
		"ghijkl",
	})
	opp.Append(opFile)

	assert.Equal(t, 2, len(opFile.Files))

	data, err := json.Marshal(opp)
	assert.NoError(t, err)

	var opp2 *OperationPack
	err = json.Unmarshal(data, &opp2)
	assert.NoError(t, err)

	ensureIds(opp)
	ensureAuthors(t, opp, opp2)

	assert.Equal(t, opp, opp2)
}

func ensureIds(opp *OperationPack) {
	for _, op := range opp.Operations {
		op.Id()
	}
}

func ensureAuthors(t *testing.T, opp1 *OperationPack, opp2 *OperationPack) {
	require.Equal(t, len(opp1.Operations), len(opp2.Operations))
	for i := 0; i < len(opp1.Operations); i++ {
		op1 := opp1.Operations[i]
		op2 := opp2.Operations[i]

		// ensure we have equivalent authors (IdentityStub vs Identity) then
		// enforce equality
		require.Equal(t, op1.base().Author.Id(), op2.base().Author.Id())
		op1.base().Author = op2.base().Author
	}
}
