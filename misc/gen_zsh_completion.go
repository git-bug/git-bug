// +build ignore

package main

import (
	"fmt"
	"log"
	"os"
	"path"

	"github.com/MichaelMure/git-bug/commands"
)

func main() {
	cwd, _ := os.Getwd()
	filepath := path.Join(cwd, "misc", "zsh_completion", "git-bug")

	fmt.Println("Generating ZSH completion file ...")

	err := commands.RootCmd.GenZshCompletionFile(filepath)
	if err != nil {
		log.Fatal(err)
	}
}
