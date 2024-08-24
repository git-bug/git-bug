package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"

	"github.com/git-bug/git-bug/commands"
)

func main() {
	fmt.Println("Generating documentation ...")

	tasks := map[string]func(*cobra.Command) error{
		"ManPage":  genManPage,
		"Markdown": genMarkdown,
	}

	var wg sync.WaitGroup
	for name, f := range tasks {
		wg.Add(1)
		go func(name string, f func(*cobra.Command) error) {
			defer wg.Done()
			root := commands.NewRootCommand()
			err := f(root)
			if err != nil {
				fmt.Printf("  - %s: %v\n", name, err)
				return
			}
			fmt.Printf("  - %s: ok\n", name)
		}(name, f)
	}

	wg.Wait()
}

func genManPage(root *cobra.Command) error {
	cwd, _ := os.Getwd()
	dir := filepath.Join(cwd, "doc", "man")

	// fixed date to avoid having to commit each month
	date := time.Date(2019, 4, 1, 12, 0, 0, 0, time.UTC)

	header := &doc.GenManHeader{
		Title:   "GIT-BUG",
		Section: "1",
		Date:    &date,
		Source:  "Generated from git-bug's source code",
	}

	files, err := filepath.Glob(dir + "/*.1")
	if err != nil {
		return err
	}
	for _, f := range files {
		if err := os.Remove(f); err != nil {
			return err
		}
	}

	return doc.GenManTree(root, header, dir)
}

func genMarkdown(root *cobra.Command) error {
	cwd, _ := os.Getwd()
	dir := filepath.Join(cwd, "doc", "md")

	files, err := filepath.Glob(dir + "/*.md")
	if err != nil {
		return err
	}
	for _, f := range files {
		if err := os.Remove(f); err != nil {
			return err
		}
	}

	return doc.GenMarkdownTree(root, dir)
}
