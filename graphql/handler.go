package graphql

import (
	"context"
	"net/http"

	"github.com/MichaelMure/git-bug/repository"
	"github.com/graphql-go/handler"
)

type Handler struct {
	Handler *handler.Handler
	Repo    repository.Repo
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := context.WithValue(r.Context(), "repo", h.Repo)
	h.Handler.ContextHandler(ctx, w, r)
}

func NewHandler(repo repository.Repo) (*Handler, error) {
	schema, err := graphqlSchema()

	if err != nil {
		return nil, err
	}

	return &Handler{
		Handler: handler.New(&handler.Config{
			Schema:   &schema,
			Pretty:   true,
			GraphiQL: true,
		}),
		Repo: repo,
	}, nil
}
