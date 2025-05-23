package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/spf13/cobra"

	"github.com/git-bug/git-bug/commands"
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
			root := commands.NewRootCommand("")
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
	f, err := os.Create(filepath.Join(cwd, "misc", "completion", "bash", "git-bug"))
	if err != nil {
		return err
	}
	defer f.Close()

	const patch = `
# Custom bash code to connect the git completion for "git bug" to the
# git-bug completion for "git-bug"
_git_bug() {
    local cur prev words cword split

    COMPREPLY=()

    # Call _init_completion from the bash-completion package
    # to prepare the arguments properly
    if declare -F _init_completion >/dev/null 2>&1; then
        _init_completion -n "=:" || return
    else
        __git-bug_init_completion -n "=:" || return
    fi

    # START PATCH
    # replace in the array ("git","bug", ...) to ("git-bug", ...) and adjust the index in cword
    words=("git-bug" "${words[@]:2}")
    cword=$(($cword-1))
    # END PATCH

    __git-bug_debug
    __git-bug_debug "========= starting completion logic =========="
    __git-bug_debug "cur is ${cur}, words[*] is ${words[*]}, #words[@] is ${#words[@]}, cword is $cword"

    # The user could have moved the cursor backwards on the command-line.
    # We need to trigger completion from the $cword location, so we need
    # to truncate the command-line ($words) up to the $cword location.
    words=("${words[@]:0:$cword+1}")
    __git-bug_debug "Truncated words[*]: ${words[*]},"

    local out directive
    __git-bug_get_completion_results
    __git-bug_process_completion_results
}
`
	err = root.GenBashCompletionV2(f, true)
	if err != nil {
		return err
	}

	// Custom bash code to connect the git completion for "git bug" to the
	// git-bug completion for "git-bug"
	_, err = f.WriteString(patch)

	return err
}

func genFish(root *cobra.Command) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	dir := filepath.Join(cwd, "misc", "completion", "fish", "git-bug")
	return root.GenFishCompletionFile(dir, true)
}

func genPowerShell(root *cobra.Command) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	path := filepath.Join(cwd, "misc", "completion", "powershell", "git-bug")
	return root.GenPowerShellCompletionFile(path)
}

func genZsh(root *cobra.Command) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	path := filepath.Join(cwd, "misc", "completion", "zsh", "git-bug")
	return root.GenZshCompletionFile(path)
}
