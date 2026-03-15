//go:generate go run doc/generate.go
//go:generate go run misc/completion/generate.go

package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/git-bug/git-bug/commands"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	v, _ := getVersion()
	root := commands.NewRootCommand(ctx, v)
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
