package graphql

import (
	"context"
	"fmt"
	"sync"

	"github.com/vektah/gqlparser/ast"
	"github.com/vektah/gqlparser/gqlerror"
)

type Resolver func(ctx context.Context) (res interface{}, err error)
type FieldMiddleware func(ctx context.Context, next Resolver) (res interface{}, err error)
type RequestMiddleware func(ctx context.Context, next func(ctx context.Context) []byte) []byte

type RequestContext struct {
	RawQuery  string
	Variables map[string]interface{}
	Doc       *ast.QueryDocument
	// ErrorPresenter will be used to generate the error
	// message from errors given to Error().
	ErrorPresenter      ErrorPresenterFunc
	Recover             RecoverFunc
	ResolverMiddleware  FieldMiddleware
	DirectiveMiddleware FieldMiddleware
	RequestMiddleware   RequestMiddleware

	errorsMu sync.Mutex
	Errors   gqlerror.List
}

func DefaultResolverMiddleware(ctx context.Context, next Resolver) (res interface{}, err error) {
	return next(ctx)
}

func DefaultDirectiveMiddleware(ctx context.Context, next Resolver) (res interface{}, err error) {
	return next(ctx)
}

func DefaultRequestMiddleware(ctx context.Context, next func(ctx context.Context) []byte) []byte {
	return next(ctx)
}

func NewRequestContext(doc *ast.QueryDocument, query string, variables map[string]interface{}) *RequestContext {
	return &RequestContext{
		Doc:                 doc,
		RawQuery:            query,
		Variables:           variables,
		ResolverMiddleware:  DefaultResolverMiddleware,
		DirectiveMiddleware: DefaultDirectiveMiddleware,
		RequestMiddleware:   DefaultRequestMiddleware,
		Recover:             DefaultRecover,
		ErrorPresenter:      DefaultErrorPresenter,
	}
}

type key string

const (
	request  key = "request_context"
	resolver key = "resolver_context"
)

func GetRequestContext(ctx context.Context) *RequestContext {
	val := ctx.Value(request)
	if val == nil {
		return nil
	}

	return val.(*RequestContext)
}

func WithRequestContext(ctx context.Context, rc *RequestContext) context.Context {
	return context.WithValue(ctx, request, rc)
}

type ResolverContext struct {
	Parent *ResolverContext
	// The name of the type this field belongs to
	Object string
	// These are the args after processing, they can be mutated in middleware to change what the resolver will get.
	Args map[string]interface{}
	// The raw field
	Field CollectedField
	// The index of array in path.
	Index *int
	// The result object of resolver
	Result interface{}
}

func (r *ResolverContext) Path() []interface{} {
	var path []interface{}
	for it := r; it != nil; it = it.Parent {
		if it.Index != nil {
			path = append(path, *it.Index)
		} else if it.Field.Field != nil {
			path = append(path, it.Field.Alias)
		}
	}

	// because we are walking up the chain, all the elements are backwards, do an inplace flip.
	for i := len(path)/2 - 1; i >= 0; i-- {
		opp := len(path) - 1 - i
		path[i], path[opp] = path[opp], path[i]
	}

	return path
}

func GetResolverContext(ctx context.Context) *ResolverContext {
	val, _ := ctx.Value(resolver).(*ResolverContext)
	return val
}

func WithResolverContext(ctx context.Context, rc *ResolverContext) context.Context {
	rc.Parent = GetResolverContext(ctx)
	return context.WithValue(ctx, resolver, rc)
}

// This is just a convenient wrapper method for CollectFields
func CollectFieldsCtx(ctx context.Context, satisfies []string) []CollectedField {
	resctx := GetResolverContext(ctx)
	return CollectFields(ctx, resctx.Field.Selections, satisfies)
}

// Errorf sends an error string to the client, passing it through the formatter.
func (c *RequestContext) Errorf(ctx context.Context, format string, args ...interface{}) {
	c.errorsMu.Lock()
	defer c.errorsMu.Unlock()

	c.Errors = append(c.Errors, c.ErrorPresenter(ctx, fmt.Errorf(format, args...)))
}

// Error sends an error to the client, passing it through the formatter.
func (c *RequestContext) Error(ctx context.Context, err error) {
	c.errorsMu.Lock()
	defer c.errorsMu.Unlock()

	c.Errors = append(c.Errors, c.ErrorPresenter(ctx, err))
}

// HasError returns true if the current field has already errored
func (c *RequestContext) HasError(rctx *ResolverContext) bool {
	c.errorsMu.Lock()
	defer c.errorsMu.Unlock()
	path := rctx.Path()

	for _, err := range c.Errors {
		if equalPath(err.Path, path) {
			return true
		}
	}
	return false
}

func equalPath(a []interface{}, b []interface{}) bool {
	if len(a) != len(b) {
		return false
	}

	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

// AddError is a convenience method for adding an error to the current response
func AddError(ctx context.Context, err error) {
	GetRequestContext(ctx).Error(ctx, err)
}

// AddErrorf is a convenience method for adding an error to the current response
func AddErrorf(ctx context.Context, format string, args ...interface{}) {
	GetRequestContext(ctx).Errorf(ctx, format, args...)
}
