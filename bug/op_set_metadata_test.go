package bug

import (
	"testing"
	"time"

	"github.com/MichaelMure/git-bug/identity"
	"github.com/stretchr/testify/assert"
)

func TestSetMetadata(t *testing.T) {
	snapshot := Snapshot{}

	var rene = identity.NewIdentity("René Descartes", "rene@descartes.fr")

	unix := time.Now().Unix()

	create := NewCreateOp(rene, unix, "title", "create", nil)
	create.SetMetadata("key", "value")
	create.Apply(&snapshot)
	snapshot.Operations = append(snapshot.Operations, create)

	hash1, err := create.Hash()
	if err != nil {
		t.Fatal(err)
	}

	comment := NewAddCommentOp(rene, unix, "comment", nil)
	comment.SetMetadata("key2", "value2")
	comment.Apply(&snapshot)
	snapshot.Operations = append(snapshot.Operations, comment)

	hash2, err := comment.Hash()
	if err != nil {
		t.Fatal(err)
	}

	op1 := NewSetMetadataOp(rene, unix, hash1, map[string]string{
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

	op2 := NewSetMetadataOp(rene, unix, hash2, map[string]string{
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

	op3 := NewSetMetadataOp(rene, unix, hash1, map[string]string{
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
