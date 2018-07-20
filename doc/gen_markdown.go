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
	filepath := path.Join(cwd, "doc", "md")

	fmt.Println("Generating Markdown documentation ...")

	err := doc.GenMarkdownTree(commands.RootCmd, filepath)
	if err != nil {
		log.Fatal(err)
	}
}
