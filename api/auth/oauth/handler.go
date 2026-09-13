package oauth

import (
	"net/http"
	"time"

	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"

	"github.com/git-bug/git-bug/api/auth"
	"github.com/git-bug/git-bug/cache"
)

// Handler handles the OAuth login/callback/logout routes.
type Handler struct {
	identities   *cache.RepoCacheIdentity
	secret       []byte
	ttl          time.Duration
	// completeAuth is the function that completes the OAuth dance and returns the
	// authenticated user. Replaced in tests to avoid real network calls.
	completeAuth func(w http.ResponseWriter, r *http.Request) (goth.User, error)
}

// NewHandler creates a new OAuth handler.
func NewHandler(identities *cache.RepoCacheIdentity, secret []byte, ttl time.Duration) *Handler {
	return &Handler{
		identities:   identities,
		secret:       secret,
		ttl:          ttl,
		completeAuth: gothic.CompleteUserAuth,
	}
}

// Begin redirects the user to the provider's authorization URL.
func (h *Handler) Begin(w http.ResponseWriter, r *http.Request) {
	gothic.BeginAuthHandler(w, r)
}

// Callback handles the provider redirect, imports the identity, and sets the
// session cookie before redirecting back to the webui root.
func (h *Handler) Callback(w http.ResponseWriter, r *http.Request) {
	gothUser, err := h.completeAuth(w, r)
	if err != nil {
		http.Error(w, "authentication failed: "+err.Error(), http.StatusUnauthorized)
		return
	}

	provider := gothUser.Provider

	identity, err := FindOrImport(h.identities, gothUser, provider)
	if err != nil {
		http.Error(w, "identity error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	token, err := auth.CreateToken(identity.Id(), h.secret, h.ttl)
	if err != nil {
		http.Error(w, "session error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, auth.SessionCookie(token, h.ttl))
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// Logout clears the session cookie.
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, auth.ExpiredCookie())
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
