package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/git-bug/git-bug/entity"
)

const DefaultSessionTTL = 24 * time.Hour

type claims struct {
	jwt.RegisteredClaims
}

// CreateToken issues a signed JWT encoding the given identity ID.
func CreateToken(id entity.Id, secret []byte, ttl time.Duration) (string, error) {
	now := time.Now()
	c := claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   id.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(secret)
}

// ValidateToken parses and validates a JWT, returning the encoded identity ID.
func ValidateToken(tokenStr string, secret []byte) (entity.Id, error) {
	tok, err := jwt.ParseWithClaims(tokenStr, &claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrNotAuthenticated
		}
		return secret, nil
	})
	if err != nil {
		return entity.UnsetId, ErrNotAuthenticated
	}
	c, ok := tok.Claims.(*claims)
	if !ok || !tok.Valid {
		return entity.UnsetId, ErrNotAuthenticated
	}
	return entity.Id(c.Subject), nil
}
