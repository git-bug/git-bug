// Package repoctx carries the per-request repository name through the
// GraphQL handler so the default-repo resolver can route to the correct
// repo in a multi-repo webui.
package repoctx

import "context"

type ctxKey struct{}

// WithName returns a copy of ctx that carries the given repository name.
func WithName(ctx context.Context, name string) context.Context {
	if name == "" {
		return ctx
	}
	return context.WithValue(ctx, ctxKey{}, name)
}

// Name returns the repo name stashed in ctx, or "" if none was set.
func Name(ctx context.Context) string {
	v, _ := ctx.Value(ctxKey{}).(string)
	return v
}
