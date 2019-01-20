package bug

import (
	"encoding/json"
	"testing"

	"github.com/MichaelMure/git-bug/util/git"
	"github.com/stretchr/testify/assert"
)

func TestOperationPackSerialize(t *testing.T) {
	opp := &OperationPack{}

	opp.Append(createOp)
	opp.Append(setTitleOp)
	opp.Append(addCommentOp)
	opp.Append(setStatusOp)
	opp.Append(labelChangeOp)

	opMeta := NewCreateOp(rene, unix, "title", "message", nil)
	opMeta.SetMetadata("key", "value")
	opp.Append(opMeta)

	assert.Equal(t, 1, len(opMeta.Metadata))

	opFile := NewCreateOp(rene, unix, "title", "message", []git.Hash{
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
	assert.Equal(t, opp, opp2)
}
