package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTokenSerial(t *testing.T) {
	original := NewToken("github", "value")
	loaded := testCredentialSerial(t, original)
	assert.Equal(t, original.Value, loaded.(*Token).Value)
}
