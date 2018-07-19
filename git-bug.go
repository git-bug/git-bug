//go:generate go run pack_webui.go

package main

import "github.com/MichaelMure/git-bug/commands"

func main() {
	commands.Execute()
}
