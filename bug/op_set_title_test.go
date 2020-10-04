package bug

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/MichaelMure/git-bug/identity"
	"github.com/MichaelMure/git-bug/repository"

	"github.com/stretchr/testify/assert"
)

func TestSetTitleSerialize(t *testing.T) {
	repo := repository.NewMockRepoForTest()
	rene := identity.NewIdentity("René Descartes", "rene@descartes.fr")
	err := rene.Commit(repo)
	require.NoError(t, err)

	unix := time.Now().Unix()
	before := NewSetTitleOp(rene, unix, "title", "was")

	data, err := json.Marshal(before)
	assert.NoError(t, err)

	var after SetTitleOperation
	err = json.Unmarshal(data, &after)
	assert.NoError(t, err)

	// enforce creating the ID
	before.Id()

	// Replace the identity stub with the real thing
	assert.Equal(t, rene.Id(), after.base().Author.Id())
	after.Author = rene

	assert.Equal(t, before, &after)
}
