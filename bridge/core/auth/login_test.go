package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoginSerial(t *testing.T) {
	original := NewLogin("github", "jean")
	loaded := testCredentialSerial(t, original)
	assert.Equal(t, original.Login, loaded.(*Login).Login)
}
