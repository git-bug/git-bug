package graphql

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/99designs/gqlgen/graphql"

	"github.com/git-bug/git-bug/util/colors"
)

// adapted from https://github.com/99designs/gqlgen/blob/master/graphql/handler/debug/tracer.go

type Tracer struct {
	Out io.Writer
}

var _ interface {
	graphql.HandlerExtension
	graphql.ResponseInterceptor
} = &Tracer{}

func (a Tracer) ExtensionName() string {
	return "error tracer"
}

func (a *Tracer) Validate(schema graphql.ExecutableSchema) error {
	return nil
}

func stringify(value interface{}) string {
	valueJson, err := json.MarshalIndent(value, "  ", "  ")
	if err == nil {
		return string(valueJson)
	}

	return fmt.Sprint(value)
}

func (a Tracer) InterceptResponse(ctx context.Context, next graphql.ResponseHandler) *graphql.Response {
	resp := next(ctx)

	if len(resp.Errors) == 0 {
		return resp
	}

	rctx := graphql.GetOperationContext(ctx)

	_, _ = fmt.Fprintln(a.Out, "GraphQL Request {")
	for _, line := range strings.Split(rctx.RawQuery, "\n") {
		_, _ = fmt.Fprintln(a.Out, " ", colors.Cyan(line))
	}
	for name, value := range rctx.Variables {
		_, _ = fmt.Fprintf(a.Out, "  var %s = %s\n", name, colors.Yellow(stringify(value)))
	}

	_, _ = fmt.Fprintln(a.Out, "  resp:", colors.Green(stringify(resp)))
	for _, err := range resp.Errors {
		_, _ = fmt.Fprintln(a.Out, "  error:", colors.Bold(err.Path.String()+":"), colors.Red(err.Message))
	}
	_, _ = fmt.Fprintln(a.Out, "}")
	_, _ = fmt.Fprintln(a.Out)
	return resp
}
