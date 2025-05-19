//go:generate go tool gqlgen generate

// Package graphql contains the root GraphQL http handler
package graphql

import (
	"io"
	"net/http"

	"github.com/99designs/gqlgen/graphql/handler"

	"github.com/git-bug/git-bug/api/graphql/graph"
	"github.com/git-bug/git-bug/api/graphql/resolvers"
	"github.com/git-bug/git-bug/cache"
)

// Handler is the root GraphQL http handler
type Handler struct {
	http.Handler
	io.Closer
}

func NewHandler(mrc *cache.MultiRepoCache, errorOut io.Writer) Handler {
	rootResolver := resolvers.NewRootResolver(mrc)
	config := graph.Config{Resolvers: rootResolver}
	h := handler.NewDefaultServer(graph.NewExecutableSchema(config))

	if errorOut != nil {
		h.Use(&Tracer{Out: errorOut})
	}

	return Handler{
		Handler: h,
		Closer:  rootResolver,
	}
}
