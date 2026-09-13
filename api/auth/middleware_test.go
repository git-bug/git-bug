package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/entity"
)

func TestJWTMiddleware_ValidToken(t *testing.T) {
	id := entity.Id("abc123def456")
	tok, err := CreateToken(id, testSecret, time.Hour)
	require.NoError(t, err)

	var capturedId entity.Id
	handler := JWTMiddleware(testSecret, time.Hour)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Resolve the raw id from context without a real repo.
		raw, ok := r.Context().Value(identityCtxKey).(entity.Id)
		if ok {
			capturedId = raw
		}
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{Name: SessionCookieName, Value: tok})
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, id, capturedId)
	// Sliding window: a fresh Set-Cookie must be present and valid.
	cookies := rr.Result().Cookies()
	var found bool
	for _, c := range cookies {
		if c.Name == SessionCookieName {
			found = true
			_, err := ValidateToken(c.Value, testSecret)
			assert.NoError(t, err, "reissued cookie must contain a valid token")
		}
	}
	assert.True(t, found, "expected Set-Cookie header")
}

func TestJWTMiddleware_NoCookie(t *testing.T) {
	var called bool
	handler := JWTMiddleware(testSecret, time.Hour)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		// Context must not have an identity.
		_, ok := r.Context().Value(identityCtxKey).(entity.Id)
		assert.False(t, ok)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.True(t, called)
	// No Set-Cookie should be issued.
	assert.Empty(t, rr.Result().Cookies())
}

func TestJWTMiddleware_ExpiredToken(t *testing.T) {
	id := entity.Id("abc123def456")
	tok, err := CreateToken(id, testSecret, -time.Second)
	require.NoError(t, err)

	var called bool
	handler := JWTMiddleware(testSecret, time.Hour)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		_, ok := r.Context().Value(identityCtxKey).(entity.Id)
		assert.False(t, ok, "expired token must not set identity in context")
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{Name: SessionCookieName, Value: tok})
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.True(t, called)
	// Stale cookie must be cleared.
	cookies := rr.Result().Cookies()
	require.Len(t, cookies, 1)
	assert.Equal(t, SessionCookieName, cookies[0].Name)
	assert.True(t, cookies[0].MaxAge < 0, "expired cookie must have negative MaxAge")
}
