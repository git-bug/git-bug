//go:generate go run doc/gen_markdown.go
//go:generate go run doc/gen_manpage.go
//go:generate go run misc/gen_bash_completion.go
//go:generate go run misc/gen_zsh_completion.go

package main

import (
	"github.com/MichaelMure/git-bug/commands"

	// minimal go version is 1.9
	_ "github.com/theckman/goconstraint/go1.9/gte"
)

func main() {
	commands.Execute()
}
