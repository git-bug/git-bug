//go:build tools

package tools

import (
	_ "github.com/99designs/gqlgen"
	_ "github.com/cheekybits/genny"
	_ "github.com/praetorian-inc/gokart"
	_ "github.com/selesy/gokart-pre"
	_ "golang.org/x/vuln/cmd/govulncheck"
)
