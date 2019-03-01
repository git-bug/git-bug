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
	dir := path.Join(cwd, "misc", "bash_completion", "git-bug")

	fmt.Println("Generating bash completion file ...")

	err := commands.RootCmd.GenBashCompletionFile(dir)
	if err != nil {
		log.Fatal(err)
	}
}
