//go:generate go run doc/generate.go
//go:generate go run misc/completion/generate.go

package main

import (
	"github.com/git-bug/git-bug/commands"
)

func main() {
	commands.Execute()
}
