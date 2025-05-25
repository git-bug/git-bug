package buginput

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/git-bug/git-bug/commands/input"
	"github.com/git-bug/git-bug/repository"
)

const messageFilename = "BUG_EDITMSG"

// ErrEmptyMessage is returned when the required message has not been entered
var ErrEmptyMessage = errors.New("empty message")

// ErrEmptyTitle is returned when the required title has not been entered
var ErrEmptyTitle = errors.New("empty title")

const bugTitleCommentTemplate = `%s%s

# Please enter the title and comment message. The first non-empty line will be
# used as the title. Lines starting with '#' will be ignored.
# An empty title aborts the operation.
`

// BugCreateEditorInput will open the default editor in the terminal with a
// template for the user to fill. The file is then processed to extract title
// and message.
func BugCreateEditorInput(repo repository.RepoCommonStorage, preTitle string, preMessage string) (string, string, error) {
	if preMessage != "" {
		preMessage = "\n\n" + preMessage
	}

	template := fmt.Sprintf(bugTitleCommentTemplate, preTitle, preMessage)

	raw, err := input.LaunchEditorWithTemplate(repo, messageFilename, template)
	if err != nil {
		return "", "", err
	}

	return processCreate(raw)
}

// BugCreateFileInput read from either from a file or from the standard input
// and extract a title and a message
func BugCreateFileInput(fileName string) (string, string, error) {
	raw, err := input.FromFile(fileName)
	if err != nil {
		return "", "", err
	}

	return processCreate(raw)
}

func processCreate(raw string) (string, string, error) {
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

const bugCommentTemplate = `%s

# Please enter the comment message. Lines starting with '#' will be ignored,
# and an empty message aborts the operation.
`

// BugCommentEditorInput will open the default editor in the terminal with a
// template for the user to fill. The file is then processed to extract a comment.
func BugCommentEditorInput(repo repository.RepoCommonStorage, preMessage string) (string, error) {
	template := fmt.Sprintf(bugCommentTemplate, preMessage)

	raw, err := input.LaunchEditorWithTemplate(repo, messageFilename, template)
	if err != nil {
		return "", err
	}

	return processComment(raw)
}

// BugCommentFileInput read from either from a file or from the standard input
// and extract a message
func BugCommentFileInput(fileName string) (string, error) {
	raw, err := input.FromFile(fileName)
	if err != nil {
		return "", err
	}

	return processComment(raw)
}

func processComment(raw string) (string, error) {
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
func BugTitleEditorInput(repo repository.RepoCommonStorage, preTitle string) (string, error) {
	template := fmt.Sprintf(bugTitleTemplate, preTitle)

	raw, err := input.LaunchEditorWithTemplate(repo, messageFilename, template)
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

const queryTemplate = `%s

# Please edit the bug query.
# Lines starting with '#' will be ignored, and an empty query aborts the operation.
#
# Example: status:open author:"rené descartes" sort:edit
#
# Valid filters are:
#
# - status:open, status:closed
# - author:<query>
# - title:<title>
# - label:<label>
# - no:label
#
# Sorting
#
# - sort:id, sort:id-desc, sort:id-asc
# - sort:creation, sort:creation-desc, sort:creation-asc
# - sort:edit, sort:edit-desc, sort:edit-asc
#
# Notes
#
# - queries are case insensitive.
# - you can combine as many qualifiers as you want.
# - you can use double quotes for multi-word search terms (ex: author:"René Descartes")
`

// QueryEditorInput will open the default editor in the terminal with a
// template for the user to fill. The file is then processed to extract a query.
func QueryEditorInput(repo repository.RepoCommonStorage, preQuery string) (string, error) {
	template := fmt.Sprintf(queryTemplate, preQuery)

	raw, err := input.LaunchEditorWithTemplate(repo, messageFilename, template)
	if err != nil {
		return "", err
	}

	lines := strings.Split(raw, "\n")

	for _, line := range lines {
		if strings.HasPrefix(line, "#") {
			continue
		}
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		return trimmed, nil
	}

	return "", nil
}
