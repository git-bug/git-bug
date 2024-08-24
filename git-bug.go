//go:generate go run doc/gen_docs.go
//go:generate go run misc/completion/gen_completion.go

package main

import (
	"github.com/git-bug/git-bug/commands"
)

func main() {
	commands.Execute()
}
