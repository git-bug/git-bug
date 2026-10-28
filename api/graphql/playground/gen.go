//go:build ignore

// Renders gqlgen's playground page into playground.html, see playground.go
// for why.
package main

import (
	"log"
	"net/http/httptest"
	"os"

	"github.com/99designs/gqlgen/graphql/playground"
)

func main() {
	rec := httptest.NewRecorder()
	playground.Handler("git-bug", "/graphql").ServeHTTP(rec, httptest.NewRequest("GET", "/playground", nil))
	if err := os.WriteFile("playground.html", rec.Body.Bytes(), 0644); err != nil {
		log.Fatal(err)
	}
}
