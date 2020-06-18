//go:generate go run gen_graphql.go

// Package graphql contains the root GraphQL http handler
package graphql

import (
	"net/http"

	"github.com/99designs/gqlgen/graphql/handler"

	"github.com/MichaelMure/git-bug/graphql/config"
	"github.com/MichaelMure/git-bug/graphql/graph"
	"github.com/MichaelMure/git-bug/graphql/resolvers"
	"github.com/MichaelMure/git-bug/repository"
)

// Handler is the root GraphQL http handler
type Handler struct {
	http.Handler
	*resolvers.RootResolver
}

func NewHandler(repo repository.ClockedRepo, cfg config.Config) (Handler, error) {
	h := Handler{
		RootResolver: resolvers.NewRootResolver(cfg),
	}

	err := h.RootResolver.RegisterDefaultRepository(repo)
	if err != nil {
		return Handler{}, err
	}

	config := graph.Config{
		Resolvers: h.RootResolver,
	}

	h.Handler = handler.NewDefaultServer(graph.NewExecutableSchema(config))

	return h, nil
}
