//go:build tools

package tools

import (
	_ "github.com/99designs/gqlgen"
	_ "github.com/praetorian-inc/gokart"
	_ "github.com/shurcooL/httpfs/filter"
	_ "github.com/shurcooL/vfsgen"
	_ "golang.org/x/vuln/cmd/govulncheck"
)
