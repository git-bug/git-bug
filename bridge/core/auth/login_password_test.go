package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoginPasswordSerial(t *testing.T) {
	original := NewLoginPassword("github", "jean", "jacques")
	loaded := testCredentialSerial(t, original)
	assert.Equal(t, original.Login, loaded.(*LoginPassword).Login)
	assert.Equal(t, original.Password, loaded.(*LoginPassword).Password)
}
