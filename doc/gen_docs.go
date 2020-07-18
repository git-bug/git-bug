package main

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/spf13/cobra/doc"

	"github.com/MichaelMure/git-bug/commands"
)

func main() {
	fmt.Println("Generating documentation ...")

	tasks := map[string]func() error{
		"BridgeConfig": commands.GenBridgeConfig,
		"ManPage":      genManPage,
		"Markdown":     genMarkdown,
	}

	// Due to concurrency issues in cobra, the following can't be concurrent :(

	// var wg sync.WaitGroup
	for name, f := range tasks {
		// wg.Add(1)
		// go func(name string, f func() error) {
		// 	defer wg.Done()
		err := f()
		if err != nil {
			fmt.Printf("  - %s: %v\n", name, err)
			return
		}
		fmt.Printf("  - %s: ok\n", name)
		// }(name, f)
	}

	// wg.Wait()
}

func genManPage() error {
	cwd, _ := os.Getwd()
	dir := path.Join(cwd, "doc", "man")

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

	return doc.GenManTree(commands.NewRootCommand(), header, dir)
}

func genMarkdown() error {
	cwd, _ := os.Getwd()
	dir := path.Join(cwd, "doc", "md")

	files, err := filepath.Glob(dir + "/*.md")
	if err != nil {
		return err
	}
	for _, f := range files {
		if err := os.Remove(f); err != nil {
			return err
		}
	}

	return doc.GenMarkdownTree(commands.NewRootCommand(), dir)
}
