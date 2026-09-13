// Package oauth handles external provider authentication and identity mapping.
// It maps incoming goth.User values to git-bug Identities, creating them on
// first login, and returns the matching Identity on subsequent logins.
package oauth

import (
	"fmt"

	"github.com/markbates/goth"

	"github.com/git-bug/git-bug/cache"
	"github.com/git-bug/git-bug/entity"
)

// metaKeyFormat is the Immutable Metadata key for a provider's stable user ID.
// e.g. "github:user-id" → "12345678"
func metaKey(provider string) string {
	return fmt.Sprintf("%s:user-id", provider)
}

// FindOrImport finds the existing Identity for the given OAuth user, or creates
// a new one on first login. The provider's numeric/opaque UserID (never the
// mutable login) is stored as Immutable Metadata so subsequent logins find the
// same Identity.
func FindOrImport(identities *cache.RepoCacheIdentity, user goth.User, provider string) (*cache.IdentityCache, error) {
	key := metaKey(provider)

	existing, err := identities.ResolveIdentityImmutableMetadata(key, user.UserID)
	if err == nil {
		return existing, nil
	}
	if !entity.IsErrNotFound(err) {
		return nil, err
	}

	return identities.NewRaw(
		user.Name,
		user.Email,
		user.NickName,
		user.AvatarURL,
		nil,
		map[string]string{key: user.UserID},
	)
}
