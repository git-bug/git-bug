package graphql

import "github.com/graphql-go/handler"

func NewHandler() (*handler.Handler, error) {
	schema, err := newGraphqlSchema()

	if err != nil {
		return nil, err
	}

	return handler.New(&handler.Config{
		Schema:   &schema,
		Pretty:   true,
		GraphiQL: true,
	}), nil
}
