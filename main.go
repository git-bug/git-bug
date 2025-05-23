//go:generate go run doc/generate.go
//go:generate go run misc/completion/generate.go

package main

import (
	"os"

	"github.com/git-bug/git-bug/commands"
)

func main() {
	v, _ := getVersion()
	root := commands.NewRootCommand(v)
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
