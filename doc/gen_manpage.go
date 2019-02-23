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
	dir := path.Join(cwd, "doc", "man")

	header := &doc.GenManHeader{
		Title:   "GIT-BUG",
		Section: "1",
		Source:  "Generated from git-bug's source code",
	}

	fmt.Println("Generating manpage ...")

	files, err := filepath.Glob(dir + "/*.1")
	if err != nil {
		log.Fatal(err)
	}
	for _, f := range files {
		if err := os.Remove(f); err != nil {
			log.Fatal(err)
		}
	}

	err = doc.GenManTree(commands.RootCmd, header, dir)
	if err != nil {
		log.Fatal(err)
	}
}
