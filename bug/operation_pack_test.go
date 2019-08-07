package bug

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/MichaelMure/git-bug/identity"
	"github.com/MichaelMure/git-bug/util/git"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOperationPackSerialize(t *testing.T) {
	opp := &OperationPack{}

	rene := identity.NewBare("René Descartes", "rene@descartes.fr")
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

	opFile := NewAddCommentOp(rene, time.Now().Unix(), "message", []git.Hash{
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

	ensureID(t, opp)

	assert.Equal(t, opp, opp2)
}

func ensureID(t *testing.T, opp *OperationPack) {
	for _, op := range opp.Operations {
		id := op.ID()
		require.True(t, IDIsValid(id))
	}
}
