//go:generate go run gen_graphql.go

// Package graphql contains the root GraphQL http handler
package graphql

import (
	"github.com/99designs/gqlgen/handler"
	"github.com/MichaelMure/git-bug/graphql/graph"
	"github.com/MichaelMure/git-bug/graphql/resolvers"
	"github.com/MichaelMure/git-bug/repository"
	"net/http"
)

// Handler is the root GraphQL http handler
type Handler struct {
	http.HandlerFunc
	*resolvers.RootResolver
}

func NewHandler(repo repository.ClockedRepo) (Handler, error) {
	h := Handler{
		RootResolver: resolvers.NewRootResolver(),
	}

	err := h.RootResolver.RegisterDefaultRepository(repo)
	if err != nil {
		return Handler{}, err
	}

	config := graph.Config{
		Resolvers: h.RootResolver,
	}

	h.HandlerFunc = handler.GraphQL(graph.NewExecutableSchema(config))

	return h, nil
}
