// +build ignore

package main

import (
	"fmt"
	"github.com/MichaelMure/git-bug/commands"
	"github.com/spf13/cobra/doc"
	"log"
	"os"
	"path"
)

func main() {
	cwd, _ := os.Getwd()
	filepath := path.Join(cwd, "doc", "man")

	header := &doc.GenManHeader{
		Title:   "MINE",
		Section: "3",
	}

	fmt.Println("Generating manpage ...")

	err := doc.GenManTree(commands.RootCmd, header, filepath)
	if err != nil {
		log.Fatal(err)
	}
}
