// Originally taken from the git-appraise project

package input

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/MichaelMure/git-bug/repository"
	"github.com/pkg/errors"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
)

const messageFilename = "BUG_MESSAGE_EDITMSG"

// ErrEmptyMessage is returned when the required message has not been entered
var ErrEmptyMessage = errors.New("empty message")

// ErrEmptyMessage is returned when the required title has not been entered
var ErrEmptyTitle = errors.New("empty title")

const bugTitleCommentTemplate = `%s%s

# Please enter the title and comment message. The first non-empty line will be
# used as the title. Lines starting with '#' will be ignored.
# An empty title aborts the operation.
`

// BugCreateEditorInput will open the default editor in the terminal with a
// template for the user to fill. The file is then processed to extract title
// and message.
func BugCreateEditorInput(repo repository.Repo, preTitle string, preMessage string) (string, string, error) {
	if preMessage != "" {
		preMessage = "\n\n" + preMessage
	}

	template := fmt.Sprintf(bugTitleCommentTemplate, preTitle, preMessage)

	raw, err := launchEditorWithTemplate(repo, messageFilename, template)

	if err != nil {
		return "", "", err
	}

	lines := strings.Split(raw, "\n")

	var title string
	var buffer bytes.Buffer
	for _, line := range lines {
		if strings.HasPrefix(line, "#") {
			continue
		}

		if title == "" {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" {
				title = trimmed
			}
			continue
		}

		buffer.WriteString(line)
		buffer.WriteString("\n")
	}

	if title == "" {
		return "", "", ErrEmptyTitle
	}

	message := strings.TrimSpace(buffer.String())

	return title, message, nil
}

const bugCommentTemplate = `

# Please enter the comment message. Lines starting with '#' will be ignored,
# and an empty message aborts the operation.
`

// BugCommentEditorInput will open the default editor in the terminal with a
// template for the user to fill. The file is then processed to extract a comment.
func BugCommentEditorInput(repo repository.Repo) (string, error) {
	raw, err := launchEditorWithTemplate(repo, messageFilename, bugCommentTemplate)

	if err != nil {
		return "", err
	}

	lines := strings.Split(raw, "\n")

	var buffer bytes.Buffer
	for _, line := range lines {
		if strings.HasPrefix(line, "#") {
			continue
		}
		buffer.WriteString(line)
		buffer.WriteString("\n")
	}

	message := strings.TrimSpace(buffer.String())

	if message == "" {
		return "", ErrEmptyMessage
	}

	return message, nil
}

const bugTitleTemplate = `%s

# Please enter the new title. Only one line will used.
# Lines starting with '#' will be ignored, and an empty title aborts the operation.
`

// BugTitleEditorInput will open the default editor in the terminal with a
// template for the user to fill. The file is then processed to extract a title.
func BugTitleEditorInput(repo repository.Repo, preTitle string) (string, error) {
	template := fmt.Sprintf(bugTitleTemplate, preTitle)
	raw, err := launchEditorWithTemplate(repo, messageFilename, template)

	if err != nil {
		return "", err
	}

	lines := strings.Split(raw, "\n")

	var title string
	for _, line := range lines {
		if strings.HasPrefix(line, "#") {
			continue
		}
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		title = trimmed
		break
	}

	if title == "" {
		return "", ErrEmptyTitle
	}

	return title, nil
}

// launchEditorWithTemplate will launch an editor as launchEditor do, but with a
// provided template.
func launchEditorWithTemplate(repo repository.Repo, fileName string, template string) (string, error) {
	path := fmt.Sprintf("%s/.git/%s", repo.GetPath(), fileName)

	err := ioutil.WriteFile(path, []byte(template), 0644)

	if err != nil {
		return "", err
	}

	return launchEditor(repo, fileName)
}

// launchEditor launches the default editor configured for the given repo. This
// method blocks until the editor command has returned.
//
// The specified filename should be a temporary file and provided as a relative path
// from the repo (e.g. "FILENAME" will be converted to ".git/FILENAME"). This file
// will be deleted after the editor is closed and its contents have been read.
//
// This method returns the text that was read from the temporary file, or
// an error if any step in the process failed.
func launchEditor(repo repository.Repo, fileName string) (string, error) {
	path := fmt.Sprintf("%s/.git/%s", repo.GetPath(), fileName)
	defer os.Remove(path)

	editor, err := repo.GetCoreEditor()
	if err != nil {
		return "", fmt.Errorf("Unable to detect default git editor: %v\n", err)
	}

	cmd, err := startInlineCommand(editor, path)
	if err != nil {
		// Running the editor directly did not work. This might mean that
		// the editor string is not a path to an executable, but rather
		// a shell command (e.g. "emacsclient --tty"). As such, we'll try
		// to run the command through bash, and if that fails, try with sh
		args := []string{"-c", fmt.Sprintf("%s %q", editor, path)}
		cmd, err = startInlineCommand("bash", args...)
		if err != nil {
			cmd, err = startInlineCommand("sh", args...)
		}
	}
	if err != nil {
		return "", fmt.Errorf("Unable to start editor: %v\n", err)
	}

	if err := cmd.Wait(); err != nil {
		return "", fmt.Errorf("Editing finished with error: %v\n", err)
	}

	output, err := ioutil.ReadFile(path)

	if err != nil {
		return "", fmt.Errorf("Error reading edited file: %v\n", err)
	}

	return string(output), err
}

// FromFile loads and returns the contents of a given file. If - is passed
// through, much like git, it will read from stdin. This can be piped data,
// unless there is a tty in which case the user will be prompted to enter a
// message.
func FromFile(fileName string) (string, error) {
	if fileName == "-" {
		stat, err := os.Stdin.Stat()
		if err != nil {
			return "", fmt.Errorf("Error reading from stdin: %v\n", err)
		}
		if (stat.Mode() & os.ModeCharDevice) == 0 {
			// There is no tty. This will allow us to read piped data instead.
			output, err := ioutil.ReadAll(os.Stdin)
			if err != nil {
				return "", fmt.Errorf("Error reading from stdin: %v\n", err)
			}
			return string(output), err
		}

		fmt.Printf("(reading comment from standard input)\n")
		var output bytes.Buffer
		s := bufio.NewScanner(os.Stdin)
		for s.Scan() {
			output.Write(s.Bytes())
			output.WriteRune('\n')
		}
		return output.String(), nil
	}

	output, err := ioutil.ReadFile(fileName)
	if err != nil {
		return "", fmt.Errorf("Error reading file: %v\n", err)
	}
	return string(output), err
}

func startInlineCommand(command string, args ...string) (*exec.Cmd, error) {
	cmd := exec.Command(command, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Start()
	return cmd, err
}
