package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/entity"
)

var testSecret = []byte("test-secret-key-32-bytes-long-xx")

func TestJWT_RoundTrip(t *testing.T) {
	id := entity.Id("abc123def456")
	tok, err := CreateToken(id, testSecret, time.Hour)
	require.NoError(t, err)
	require.NotEmpty(t, tok)

	got, err := ValidateToken(tok, testSecret)
	require.NoError(t, err)
	assert.Equal(t, id, got)
}

func TestJWT_ExpiredToken(t *testing.T) {
	id := entity.Id("abc123def456")
	tok, err := CreateToken(id, testSecret, -time.Second)
	require.NoError(t, err)

	_, err = ValidateToken(tok, testSecret)
	assert.ErrorIs(t, err, ErrNotAuthenticated)
}

func TestJWT_WrongSecret(t *testing.T) {
	id := entity.Id("abc123def456")
	tok, err := CreateToken(id, testSecret, time.Hour)
	require.NoError(t, err)

	_, err = ValidateToken(tok, []byte("wrong-secret"))
	assert.ErrorIs(t, err, ErrNotAuthenticated)
}

func TestJWT_Malformed(t *testing.T) {
	_, err := ValidateToken("not.a.jwt", testSecret)
	assert.ErrorIs(t, err, ErrNotAuthenticated)

	_, err = ValidateToken("", testSecret)
	assert.ErrorIs(t, err, ErrNotAuthenticated)
}
