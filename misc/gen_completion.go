package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/commands"
)

func main() {
	fmt.Println("Generating completion files ...")

	tasks := map[string]func(*cobra.Command) error{
		"Bash":       genBash,
		"Fish":       genFish,
		"PowerShell": genPowerShell,
		"ZSH":        genZsh,
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

func genBash(root *cobra.Command) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	f, err := os.Create(filepath.Join(cwd, "misc", "bash_completion", "git-bug"))
	if err != nil {
		return err
	}
	defer f.Close()

	// Custom bash code to connect the git completion for "git bug" to the
	// git-bug completion for "git-bug"
	_, err = f.WriteString(`#TODO: completion code to map "git bug" to "git-bug"`)
	if err != nil {
		return err
	}

	return root.GenBashCompletionV2(f, true)
}

func genFish(root *cobra.Command) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	dir := filepath.Join(cwd, "misc", "fish_completion", "git-bug")
	return root.GenFishCompletionFile(dir, true)
}

func genPowerShell(root *cobra.Command) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	path := filepath.Join(cwd, "misc", "powershell_completion", "git-bug")
	return root.GenPowerShellCompletionFile(path)
}

func genZsh(root *cobra.Command) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	path := filepath.Join(cwd, "misc", "zsh_completion", "git-bug")
	return root.GenZshCompletionFile(path)
}
