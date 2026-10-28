//go:generate go run gen.go

// Package playground serves the GraphiQL playground, a static copy of the page
// rendered by gqlgen's playground.Handler.
//
// Why a static copy: gqlgen renders the page with html/template, and executing
// a template looks up methods by name at runtime (reflect.Value.MethodByName).
// When such a call is reachable, the linker can't tell which methods might be
// called, so it keeps every exported method of every reachable type, for the
// whole binary. That alone cost ~7MB (~22%) of the release binary.
//
// Why generated: gen.go renders gqlgen's page into playground.html, so the page
// and its SRI hashes follow gqlgen updates instead of being maintained by hand.
// gen.go is only run by go generate, so gqlgen's template never reaches the
// binary.
package playground

import (
	_ "embed"
	"net/http"
)

//go:embed playground.html
var page []byte

// NewHandler serves the playground, querying /graphql.
func NewHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "text/html; charset=UTF-8")
		_, _ = w.Write(page)
	})
}
