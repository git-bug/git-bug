package auth

import (
	"net/http"

	"github.com/git-bug/git-bug/entity"
)

func Middleware(fixedUserId entity.Id) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := CtxWithUser(r.Context(), fixedUserId)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
