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
	dir := path.Join(cwd, "misc", "fish_completion", "git-bug")

	fmt.Println("Generating Fish completion file ...")

	err := commands.RootCmd.GenFishCompletionFile(dir, true)
	if err != nil {
		log.Fatal(err)
	}
}
