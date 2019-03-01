// +build ignore

package main

import (
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"

	"github.com/MichaelMure/git-bug/commands"
	"github.com/spf13/cobra/doc"
)

func main() {
	cwd, _ := os.Getwd()
	dir := path.Join(cwd, "doc", "md")

	fmt.Println("Generating Markdown documentation ...")

	files, err := filepath.Glob(dir + "/*.md")
	if err != nil {
		log.Fatal(err)
	}
	for _, f := range files {
		if err := os.Remove(f); err != nil {
			log.Fatal(err)
		}
	}

	err = doc.GenMarkdownTree(commands.RootCmd, dir)
	if err != nil {
		log.Fatal(err)
	}
}
