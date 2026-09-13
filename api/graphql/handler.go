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
	coderws "github.com/coder/websocket"
	"github.com/vektah/gqlparser/v2/ast"

	"github.com/git-bug/git-bug/api/graphql/graph"
	"github.com/git-bug/git-bug/api/graphql/resolvers"
	"github.com/git-bug/git-bug/cache"
)

func NewHandler(mrc *cache.MultiRepoCache, errorOut io.Writer, devMode bool) http.Handler {
	rootResolver := resolvers.NewRootResolver(mrc)
	config := graph.Config{Resolvers: rootResolver}

	h := handler.New(graph.NewExecutableSchema(config))

	// coder/websocket authorizes the request host itself and rejects every other
	// origin, which is the check we want in production.
	wsAcceptOptions := coderws.AcceptOptions{}
	if devMode {
		// In dev mode the Vite proxy sits on a different port than the backend,
		// so also accept loopback origins whatever their port. Patterns are
		// matched with path.Match, hence the escaping around the IPv6 literal.
		wsAcceptOptions.OriginPatterns = []string{
			"localhost", "localhost:*",
			"127.0.0.1", "127.0.0.1:*",
			`\[::1\]`, `\[::1\]:*`,
		}
	}
	h.AddTransport(transport.Websocket{
		KeepAlivePingInterval: 10 * time.Second,
		Implementation: transport.CoderWebsocketImplementation{
			AcceptOptions: wsAcceptOptions,
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
