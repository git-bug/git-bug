//go:generate go tool gqlgen generate

// Package graphql contains the root GraphQL http handler
package graphql

import (
	"io"
	"net/http"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/gorilla/websocket"
	"github.com/vektah/gqlparser/v2/ast"

	"github.com/git-bug/git-bug/api/graphql/graph"
	"github.com/git-bug/git-bug/api/graphql/resolvers"
	"github.com/git-bug/git-bug/cache"
)

func NewHandler(mrc *cache.MultiRepoCache, errorOut io.Writer) http.Handler {
	rootResolver := resolvers.NewRootResolver(mrc)
	config := graph.Config{Resolvers: rootResolver}

	h := handler.New(graph.NewExecutableSchema(config))

	h.AddTransport(transport.Websocket{
		KeepAlivePingInterval: 10 * time.Second,
		Upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	})
	h.AddTransport(transport.Options{})
	h.AddTransport(transport.GET{})
	h.AddTransport(transport.POST{})
	h.AddTransport(transport.MultipartForm{})

	h.SetQueryCache(lru.New[*ast.QueryDocument](1000))

	h.Use(extension.Introspection{})
	h.Use(extension.AutomaticPersistedQuery{
		Cache: lru.New[string](100),
	})

	if errorOut != nil {
		h.Use(&Tracer{Out: errorOut})
	}

	return h
}
