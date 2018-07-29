package graphql

import (
	"context"
)

// Error is the standard graphql error type described in https://facebook.github.io/graphql/draft/#sec-Errors
type Error struct {
	Message    string                 `json:"message"`
	Path       []interface{}          `json:"path,omitempty"`
	Locations  []ErrorLocation        `json:"locations,omitempty"`
	Extensions map[string]interface{} `json:"extensions,omitempty"`
}

func (e *Error) Error() string {
	return e.Message
}

type ErrorLocation struct {
	Line   int `json:"line,omitempty"`
	Column int `json:"column,omitempty"`
}

type ErrorPresenterFunc func(context.Context, error) *Error

type ExtendedError interface {
	Extensions() map[string]interface{}
}

func DefaultErrorPresenter(ctx context.Context, err error) *Error {
	if gqlerr, ok := err.(*Error); ok {
		gqlerr.Path = GetResolverContext(ctx).Path
		return gqlerr
	}

	var extensions map[string]interface{}
	if ee, ok := err.(ExtendedError); ok {
		extensions = ee.Extensions()
	}

	return &Error{
		Message:    err.Error(),
		Path:       GetResolverContext(ctx).Path,
		Extensions: extensions,
	}
}
