package bug

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/MichaelMure/git-bug/identity"
)

func TestAddCommentSerialize(t *testing.T) {
	var rene = identity.NewBare("René Descartes", "rene@descartes.fr")
	unix := time.Now().Unix()
	before := NewAddCommentOp(rene, unix, "message", nil)

	data, err := json.Marshal(before)
	assert.NoError(t, err)

	var after AddCommentOperation
	err = json.Unmarshal(data, &after)
	assert.NoError(t, err)

	// enforce creating the IDs
	before.Id()
	rene.Id()

	assert.Equal(t, before, &after)
}
