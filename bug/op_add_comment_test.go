package bug

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/MichaelMure/git-bug/identity"
	"github.com/stretchr/testify/assert"
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

	// enforce creating the ID
	before.ID()

	assert.Equal(t, before, &after)
}
