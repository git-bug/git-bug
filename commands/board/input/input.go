package buginput

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/git-bug/git-bug/commands/input"
	"github.com/git-bug/git-bug/repository"
)

const messageFilename = "BOARD_EDITMSG"

// ErrEmptyTitle is returned when the required title has not been entered
var ErrEmptyTitle = errors.New("empty title")

const boardTitleTemplate = `%s

# Please enter the title of the draft board item. Only one line will used.
# Lines starting with '#' will be ignored, and an empty title aborts the operation.
`

// BoardTitleEditorInput will open the default editor in the terminal with a
// template for the user to fill. The file is then processed to extract the title.
func BoardTitleEditorInput(repo repository.RepoCommonStorage, preTitle string) (string, error) {
	template := fmt.Sprintf(boardTitleTemplate, preTitle)

	raw, err := input.LaunchEditorWithTemplate(repo, messageFilename, template)
	if err != nil {
		return "", err
	}

	return processTitle(raw)
}

// BoardTitleFileInput read from either from a file or from the standard input
// and extract a title.
func BoardTitleFileInput(fileName string) (string, error) {
	raw, err := input.FromFile(fileName)
	if err != nil {
		return "", err
	}

	return processTitle(raw)
}

func processTitle(raw string) (string, error) {
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
