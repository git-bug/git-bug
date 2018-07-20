package main

import (
	"fmt"
	"github.com/MichaelMure/git-bug/commands"
	"log"
	"os"
	"path"
)

func main() {
	cwd, _ := os.Getwd()
	filepath := path.Join(cwd, "doc", "zsh_completion", "git-bug")

	fmt.Println("Generating zsh completion file ...")

	err := commands.RootCmd.GenZshCompletionFile(filepath)
	if err != nil {
		log.Fatal(err)
	}
}
