package bug

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/MichaelMure/git-bug/identity"
	"github.com/MichaelMure/git-bug/repository"

	"github.com/stretchr/testify/require"
)

func TestSetMetadata(t *testing.T) {
	snapshot := Snapshot{}

	repo := repository.NewMockRepo()

	rene, err := identity.NewIdentity(repo, "René Descartes", "rene@descartes.fr")
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
	require.Len(t, createMetadata, 2)
	// original key is not overrided
	require.Equal(t, createMetadata["key"], "value")
	// new key is set
	require.Equal(t, createMetadata["key2"], "value")

	commentMetadata := snapshot.Operations[1].AllMetadata()
	require.Len(t, commentMetadata, 1)
	require.Equal(t, commentMetadata["key2"], "value2")

	op2 := NewSetMetadataOp(rene, unix, id2, map[string]string{
		"key2": "value",
		"key3": "value3",
	})

	op2.Apply(&snapshot)
	snapshot.Operations = append(snapshot.Operations, op2)

	createMetadata = snapshot.Operations[0].AllMetadata()
	require.Len(t, createMetadata, 2)
	require.Equal(t, createMetadata["key"], "value")
	require.Equal(t, createMetadata["key2"], "value")

	commentMetadata = snapshot.Operations[1].AllMetadata()
	require.Len(t, commentMetadata, 2)
	// original key is not overrided
	require.Equal(t, commentMetadata["key2"], "value2")
	// new key is set
	require.Equal(t, commentMetadata["key3"], "value3")

	op3 := NewSetMetadataOp(rene, unix, id1, map[string]string{
		"key":  "override",
		"key2": "override",
	})

	op3.Apply(&snapshot)
	snapshot.Operations = append(snapshot.Operations, op3)

	createMetadata = snapshot.Operations[0].AllMetadata()
	require.Len(t, createMetadata, 2)
	// original key is not overrided
	require.Equal(t, createMetadata["key"], "value")
	// previously set key is not overrided
	require.Equal(t, createMetadata["key2"], "value")

	commentMetadata = snapshot.Operations[1].AllMetadata()
	require.Len(t, commentMetadata, 2)
	require.Equal(t, commentMetadata["key2"], "value2")
	require.Equal(t, commentMetadata["key3"], "value3")
}

func TestSetMetadataSerialize(t *testing.T) {
	repo := repository.NewMockRepo()

	rene, err := identity.NewIdentity(repo, "René Descartes", "rene@descartes.fr")
	require.NoError(t, err)

	unix := time.Now().Unix()
	before := NewSetMetadataOp(rene, unix, "message", map[string]string{
		"key1": "value1",
		"key2": "value2",
	})

	data, err := json.Marshal(before)
	require.NoError(t, err)

	var after SetMetadataOperation
	err = json.Unmarshal(data, &after)
	require.NoError(t, err)

	// enforce creating the ID
	before.Id()

	// Replace the identity stub with the real thing
	require.Equal(t, rene.Id(), after.base().Author.Id())
	after.Author = rene

	require.Equal(t, before, &after)
}
