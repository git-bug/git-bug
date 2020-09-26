package repository

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

// gitCli is a helper to launch CLI git commands
type gitCli struct {
	path string
}

// Run the given git command with the given I/O reader/writers, returning an error if it fails.
func (cli gitCli) runGitCommandWithIO(stdin io.Reader, stdout, stderr io.Writer, args ...string) error {
	// make sure that the working directory for the command
	// always exist, in particular when running "git init".
	path := strings.TrimSuffix(cli.path, ".git")

	// fmt.Printf("[%s] Running git %s\n", path, strings.Join(args, " "))

	cmd := exec.Command("git", args...)
	cmd.Dir = path
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	return cmd.Run()
}

// Run the given git command and return its stdout, or an error if the command fails.
func (cli gitCli) runGitCommandRaw(stdin io.Reader, args ...string) (string, string, error) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := cli.runGitCommandWithIO(stdin, &stdout, &stderr, args...)
	return strings.TrimSpace(stdout.String()), strings.TrimSpace(stderr.String()), err
}

// Run the given git command and return its stdout, or an error if the command fails.
func (cli gitCli) runGitCommandWithStdin(stdin io.Reader, args ...string) (string, error) {
	stdout, stderr, err := cli.runGitCommandRaw(stdin, args...)
	if err != nil {
		if stderr == "" {
			stderr = "Error running git command: " + strings.Join(args, " ")
		}
		err = fmt.Errorf(stderr)
	}
	return stdout, err
}

// Run the given git command and return its stdout, or an error if the command fails.
func (cli gitCli) runGitCommand(args ...string) (string, error) {
	return cli.runGitCommandWithStdin(nil, args...)
}
