//go:build tools

package tools

import (
	_ "github.com/shurcooL/httpfs/filter"
	_ "github.com/shurcooL/vfsgen"
	_ "golang.org/x/vuln/cmd/govulncheck"
)
