package bug

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/MichaelMure/git-bug/identity"
	"github.com/MichaelMure/git-bug/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetMetadata(t *testing.T) {
	snapshot := Snapshot{}

	repo := repository.NewMockRepoForTest()
	rene := identity.NewIdentity("René Descartes", "rene@descartes.fr")
	err := rene.Commit(repo)
	require.NoError(t, err)

	unix := time.Now().Unix()

	create := NewCreateOp(rene, unix, "title", "create", nil)
	create.SetMetadata("key", "value")
	create.Apply(&snapshot)
	snapshot.Operations = append(snapshot.Operations, create)

	id1 := create.Id()
	require.NoError(t, id1.Validate())

	comment := NewAddCommentOp(rene, unix, "comment", nil)
	comment.SetMetadata("key2", "value2")
	comment.Apply(&snapshot)
	snapshot.Operations = append(snapshot.Operations, comment)

	id2 := comment.Id()
	require.NoError(t, id2.Validate())

	op1 := NewSetMetadataOp(rene, unix, id1, map[string]string{
		"key":  "override",
		"key2": "value",
	})

	op1.Apply(&snapshot)
	snapshot.Operations = append(snapshot.Operations, op1)

	createMetadata := snapshot.Operations[0].AllMetadata()
	assert.Equal(t, len(createMetadata), 2)
	// original key is not overrided
	assert.Equal(t, createMetadata["key"], "value")
	// new key is set
	assert.Equal(t, createMetadata["key2"], "value")

	commentMetadata := snapshot.Operations[1].AllMetadata()
	assert.Equal(t, len(commentMetadata), 1)
	assert.Equal(t, commentMetadata["key2"], "value2")

	op2 := NewSetMetadataOp(rene, unix, id2, map[string]string{
		"key2": "value",
		"key3": "value3",
	})

	op2.Apply(&snapshot)
	snapshot.Operations = append(snapshot.Operations, op2)

	createMetadata = snapshot.Operations[0].AllMetadata()
	assert.Equal(t, len(createMetadata), 2)
	assert.Equal(t, createMetadata["key"], "value")
	assert.Equal(t, createMetadata["key2"], "value")

	commentMetadata = snapshot.Operations[1].AllMetadata()
	assert.Equal(t, len(commentMetadata), 2)
	// original key is not overrided
	assert.Equal(t, commentMetadata["key2"], "value2")
	// new key is set
	assert.Equal(t, commentMetadata["key3"], "value3")

	op3 := NewSetMetadataOp(rene, unix, id1, map[string]string{
		"key":  "override",
		"key2": "override",
	})

	op3.Apply(&snapshot)
	snapshot.Operations = append(snapshot.Operations, op3)

	createMetadata = snapshot.Operations[0].AllMetadata()
	assert.Equal(t, len(createMetadata), 2)
	// original key is not overrided
	assert.Equal(t, createMetadata["key"], "value")
	// previously set key is not overrided
	assert.Equal(t, createMetadata["key2"], "value")

	commentMetadata = snapshot.Operations[1].AllMetadata()
	assert.Equal(t, len(commentMetadata), 2)
	assert.Equal(t, commentMetadata["key2"], "value2")
	assert.Equal(t, commentMetadata["key3"], "value3")
}

func TestSetMetadataSerialize(t *testing.T) {
	repo := repository.NewMockRepoForTest()
	rene := identity.NewIdentity("René Descartes", "rene@descartes.fr")
	err := rene.Commit(repo)
	require.NoError(t, err)

	unix := time.Now().Unix()
	before := NewSetMetadataOp(rene, unix, "message", map[string]string{
		"key1": "value1",
		"key2": "value2",
	})

	data, err := json.Marshal(before)
	assert.NoError(t, err)

	var after SetMetadataOperation
	err = json.Unmarshal(data, &after)
	assert.NoError(t, err)

	// enforce creating the ID
	before.Id()

	// Replace the identity stub with the real thing
	assert.Equal(t, rene.Id(), after.base().Author.Id())
	after.Author = rene

	assert.Equal(t, before, &after)
}
