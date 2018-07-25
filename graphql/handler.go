package graphql

import (
	"context"
	"net/http"

	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/repository"
	graphqlHandler "github.com/graphql-go/handler"
)

type Handler struct {
	handler *graphqlHandler.Handler
	cache   cache.Cache
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := context.WithValue(r.Context(), "cache", h.cache)
	h.handler.ContextHandler(ctx, w, r)
}

func NewHandler(repo repository.Repo) (*Handler, error) {
	schema, err := graphqlSchema()

	if err != nil {
		return nil, err
	}

	h := graphqlHandler.New(&graphqlHandler.Config{
		Schema:   &schema,
		Pretty:   true,
		GraphiQL: true,
	})

	c := cache.NewDefaultCache()
	c.RegisterDefaultRepository(repo)

	return &Handler{
		handler: h,
		cache:   c,
	}, nil
}
