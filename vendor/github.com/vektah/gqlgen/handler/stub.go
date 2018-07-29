package handler

import (
	"context"
	"time"

	"github.com/vektah/gqlgen/graphql"
	"github.com/vektah/gqlgen/neelance/query"
	"github.com/vektah/gqlgen/neelance/schema"
)

type executableSchemaStub struct {
}

var _ graphql.ExecutableSchema = &executableSchemaStub{}

func (e *executableSchemaStub) Schema() *schema.Schema {
	return schema.MustParse(`
		schema { query: Query }
		type Query { me: User! }
		type User { name: String! }
	`)
}

func (e *executableSchemaStub) Query(ctx context.Context, op *query.Operation) *graphql.Response {
	return &graphql.Response{Data: []byte(`{"name":"test"}`)}
}

func (e *executableSchemaStub) Mutation(ctx context.Context, op *query.Operation) *graphql.Response {
	return graphql.ErrorResponse(ctx, "mutations are not supported")
}

func (e *executableSchemaStub) Subscription(ctx context.Context, op *query.Operation) func() *graphql.Response {
	return func() *graphql.Response {
		time.Sleep(50 * time.Millisecond)
		select {
		case <-ctx.Done():
			return nil
		default:
			return &graphql.Response{
				Data: []byte(`{"name":"test"}`),
			}
		}
	}
}
