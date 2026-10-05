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
	// The first signal cancels ctx, for the command to stop gracefully. The
	// default behavior is then restored: a second one kills the process,
	// whatever the command does.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		stop()
	}()

	v := getVersion()
	root := commands.NewRootCommand(ctx, v)
	if err := root.ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}
