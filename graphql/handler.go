//go:generate go run gen_graphql.go

package graphql

import (
	"github.com/MichaelMure/git-bug/graphql/graph"
	"github.com/MichaelMure/git-bug/graphql/resolvers"
	"github.com/MichaelMure/git-bug/repository"
	"github.com/vektah/gqlgen/handler"
	"net/http"
)

type Handler struct {
	http.HandlerFunc
	*resolvers.Backend
}

func NewHandler(repo repository.Repo) (Handler, error) {
	h := Handler{
		Backend: resolvers.NewBackend(),
	}

	err := h.Backend.RegisterDefaultRepository(repo)
	if err != nil {
		return Handler{}, err
	}

	h.HandlerFunc = handler.GraphQL(graph.NewExecutableSchema(h.Backend))

	return h, nil
}
