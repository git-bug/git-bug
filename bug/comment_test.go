package bug

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCompileUnpackCommentId(t *testing.T) {
	id1 := "abcdefghijklmnopqrstuvwxyz1234"
	id2 := "ABCDEFGHIJ"
	expectedId := "aAbBcCdefDghijEklmnFopqrGstuvHwxyzI1234J"

	compiledId := CompileCommentId(id1, id2)
	assert.Equal(t, expectedId, compiledId)

	unpackedId1, unpackedId2 := UnpackCommentId(compiledId)
	assert.Equal(t, id1, unpackedId1)
	assert.Equal(t, id2, unpackedId2)

	unpackedId1, unpackedId2 = UnpackCommentId(expectedId[:6])
	assert.Equal(t, unpackedId1, id1[:3])
	assert.Equal(t, unpackedId2, id2[:3])
}
