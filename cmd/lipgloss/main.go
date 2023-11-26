package main

import (
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/truncate"
	"github.com/muesli/termenv"
)

func main() {
	const (
		idText            = "9f3e262"
		statusText        = "closed"
		titleText         = "Error: the repository you want to access is already locked"
		labelIndicator    = "◼"
		authorText        = "Arnaud LE CAM (arnaudlecam)"
		commentText       = "9"
		commentIndicator  = "💬"
		ellipsisIndicator = "…"
	)

	const (
		cyan    = termenv.ANSICyan
		yellow  = termenv.ANSIYellow
		magenta = termenv.ANSIMagenta
		gray    = termenv.ANSIWhite
	)

	const (
		idLen      = 7
		statusLen  = len("closed")
		titleLen   = 50
		authorLen  = 15
		commentLen = 3
	)

	title, author := titleText, authorText
	if termenv.DefaultOutput().Profile != termenv.Ascii {
		title = truncate.StringWithTail(titleText, titleLen, ellipsisIndicator)
		author = truncate.StringWithTail(authorText, authorLen, ellipsisIndicator)
	}

	idStyle := lipgloss.NewStyle().Width(idLen).MarginRight(1).Foreground(lipgloss.ANSIColor(cyan))
	statusStyle := lipgloss.NewStyle().Width(statusLen).MarginRight(2).Foreground(lipgloss.ANSIColor(yellow))
	titleStyle := lipgloss.NewStyle().Width(titleLen).MarginRight(6).Foreground(lipgloss.ANSIColor(gray))
	authorStyle := lipgloss.NewStyle().Width(authorLen).MarginRight(1).Foreground(lipgloss.ANSIColor(magenta))
	commentStyle := lipgloss.NewStyle().Width(commentLen).MarginRight(1).Foreground(lipgloss.ANSIColor(gray)).Align(lipgloss.Right)
	commentMarker := lipgloss.NewStyle().Width(2).MarginRight(1).Foreground(lipgloss.ANSIColor(gray))

	str := strings.Join(
		[]string{
			idStyle.Render(idText),
			statusStyle.Render(statusText),
			titleStyle.Render(title),
			authorStyle.Render(author),
			commentStyle.Render(commentText),
			commentMarker.Render(commentIndicator),
			"\n",
		}, "")

	os.Stderr.Write([]byte(str))
}
