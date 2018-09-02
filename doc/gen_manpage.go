// +build ignore

package main

import (
	"fmt"
	"log"
	"os"
	"path"

	"github.com/MichaelMure/git-bug/commands"
	"github.com/spf13/cobra/doc"
)

func main() {
	cwd, _ := os.Getwd()
	filepath := path.Join(cwd, "doc", "man")

	header := &doc.GenManHeader{
		Title:   "GIT-BUG",
		Section: "1",
		Source:  "Generated from git-bug's source code",
	}

	fmt.Println("Generating manpage ...")

	err := doc.GenManTree(commands.RootCmd, header, filepath)
	if err != nil {
		log.Fatal(err)
	}
}
