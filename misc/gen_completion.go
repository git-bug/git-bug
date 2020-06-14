package main

import (
	"fmt"
	"os"
	"path"
	"sync"

	"github.com/MichaelMure/git-bug/commands"
)

func main() {
	fmt.Println("Generating completion files ...")

	tasks := map[string]func() error{
		"Bash":       genBash,
		"Fish":       genFish,
		"PowerShell": genPowerShell,
		"ZSH":        genZsh,
	}

	var wg sync.WaitGroup
	for name, f := range tasks {
		wg.Add(1)
		go func(name string, f func() error) {
			defer wg.Done()
			err := f()
			if err != nil {
				fmt.Printf("  - %s: %v\n", name, err)
				return
			}
			fmt.Printf("  - %s: ok\n", name)
		}(name, f)
	}

	wg.Wait()
}

func genBash() error {
	cwd, _ := os.Getwd()
	dir := path.Join(cwd, "misc", "bash_completion", "git-bug")
	return commands.RootCmd.GenBashCompletionFile(dir)
}

func genFish() error {
	cwd, _ := os.Getwd()
	dir := path.Join(cwd, "misc", "fish_completion", "git-bug")
	return commands.RootCmd.GenFishCompletionFile(dir, true)
}

func genPowerShell() error {
	cwd, _ := os.Getwd()
	filepath := path.Join(cwd, "misc", "powershell_completion", "git-bug")
	return commands.RootCmd.GenPowerShellCompletionFile(filepath)
}

func genZsh() error {
	cwd, _ := os.Getwd()
	filepath := path.Join(cwd, "misc", "zsh_completion", "git-bug")
	return commands.RootCmd.GenZshCompletionFile(filepath)
}
