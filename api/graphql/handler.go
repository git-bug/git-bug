//go:generate go run gen_graphql.go

// Package graphql contains the root GraphQL http handler
package graphql

import (
	"io"
	"net/http"

	"github.com/99designs/gqlgen/graphql/handler"

	"github.com/MichaelMure/git-bug/api/graphql/graph"
	"github.com/MichaelMure/git-bug/api/graphql/resolvers"
	"github.com/MichaelMure/git-bug/cache"
)

// Handler is the root GraphQL http handler
type Handler struct {
	http.Handler
	io.Closer
}

func NewHandler(mrc *cache.MultiRepoCache) Handler {
	rootResolver := resolvers.NewRootResolver(mrc)
	config := graph.Config{Resolvers: rootResolver}
	h := handler.NewDefaultServer(graph.NewExecutableSchema(config))

	return Handler{
		Handler: h,
		Closer:  rootResolver,
	}
}
