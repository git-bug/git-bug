//go:generate gorunpkg github.com/vektah/gqlgen

package graphql

import (
	"github.com/MichaelMure/git-bug/graphql/resolvers"
	"github.com/MichaelMure/git-bug/repository"
	"github.com/vektah/gqlgen/handler"
	"net/http"
)

func NewHandler(repo repository.Repo) http.Handler {
	backend := resolvers.NewRootResolver()

	backend.RegisterDefaultRepository(repo)

	return handler.GraphQL(resolvers.NewExecutableSchema(backend))
}
