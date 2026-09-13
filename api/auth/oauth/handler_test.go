package oauth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/markbates/goth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/api/auth"
)

var testSecret = []byte("test-secret-key-32-bytes-long-xx")

func makeHandler(t *testing.T) (*Handler, func(goth.User, error)) {
	t.Helper()
	identities := newTestCache(t)

	var stubUser goth.User
	var stubErr error
	h := NewHandler(identities, testSecret, time.Hour)
	h.completeAuth = func(w http.ResponseWriter, r *http.Request) (goth.User, error) {
		return stubUser, stubErr
	}

	setStub := func(u goth.User, err error) {
		stubUser = u
		stubErr = err
	}
	return h, setStub
}

func TestCallback_SetsSessionCookieAndRedirects(t *testing.T) {
	h, setStub := makeHandler(t)
	setStub(makeGothUser("12345678", "Alice", "alice@example.com", "alice", ""), nil)

	req := httptest.NewRequest("GET", "/auth/github/callback?code=fakecode", nil)
	rr := httptest.NewRecorder()
	h.Callback(rr, req)

	assert.Equal(t, http.StatusSeeOther, rr.Code)
	assert.Equal(t, "/", rr.Header().Get("Location"))

	cookies := rr.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == auth.SessionCookieName {
			sessionCookie = c
		}
	}
	require.NotNil(t, sessionCookie, "session cookie must be set")
	assert.True(t, sessionCookie.HttpOnly)
	assert.NotEmpty(t, sessionCookie.Value)

	// Token must decode to a valid identity ID.
	id, err := auth.ValidateToken(sessionCookie.Value, testSecret)
	require.NoError(t, err)
	assert.NotEmpty(t, id)
}

func TestCallback_ProviderErrorReturns401(t *testing.T) {
	h, setStub := makeHandler(t)
	setStub(goth.User{}, assert.AnError)

	req := httptest.NewRequest("GET", "/auth/github/callback", nil)
	rr := httptest.NewRecorder()
	h.Callback(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.Empty(t, rr.Result().Cookies())
}

func TestLogout_ClearsSessionCookie(t *testing.T) {
	h, _ := makeHandler(t)

	req := httptest.NewRequest("POST", "/auth/logout", nil)
	rr := httptest.NewRecorder()
	h.Logout(rr, req)

	assert.Equal(t, http.StatusSeeOther, rr.Code)
	cookies := rr.Result().Cookies()
	require.Len(t, cookies, 1)
	assert.Equal(t, auth.SessionCookieName, cookies[0].Name)
	assert.True(t, cookies[0].MaxAge < 0, "logout must expire the session cookie")
}
