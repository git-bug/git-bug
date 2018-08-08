// +build ignore

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
	filepath := path.Join(cwd, "misc", "bash_completion", "git-bug")

	fmt.Println("Generating bash completion file ...")

	err := commands.RootCmd.GenBashCompletionFile(filepath)
	if err != nil {
		log.Fatal(err)
	}
}
