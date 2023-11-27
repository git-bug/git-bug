package main

import (
	"flag"
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

	idStyle := lipgloss.NewStyle()
	statusStyle := lipgloss.NewStyle()
	titleStyle := lipgloss.NewStyle()
	authorStyle := lipgloss.NewStyle()
	commentStyle := lipgloss.NewStyle()
	commentMarker := lipgloss.NewStyle()

	separator := "\t"

	title, author := titleText, authorText
	if termenv.DefaultOutput().Profile != termenv.Ascii {
		title = truncate.StringWithTail(titleText, titleLen, ellipsisIndicator)
		author = truncate.StringWithTail(authorText, authorLen, ellipsisIndicator)

		idStyle = idStyle.Width(idLen).MarginRight(1).Foreground(lipgloss.ANSIColor(cyan))
		statusStyle = statusStyle.Width(statusLen).MarginRight(2).Foreground(lipgloss.ANSIColor(yellow))
		titleStyle = titleStyle.Width(titleLen).MarginRight(6).Foreground(lipgloss.ANSIColor(gray))
		authorStyle = authorStyle.Width(authorLen).MarginRight(1).Foreground(lipgloss.ANSIColor(magenta))
		commentStyle = commentStyle.Width(commentLen).MarginRight(1).Foreground(lipgloss.ANSIColor(gray)).Align(lipgloss.Right)
		commentMarker = commentMarker.Width(2).MarginRight(1).Foreground(lipgloss.ANSIColor(gray))

		separator = ""
	}

	type formatFunc func() string

	defaultFormatFunc := func() string {
		return strings.Join(
			[]string{
				idStyle.Render(idText),
				statusStyle.Render(statusText),
				titleStyle.Render(title),
				authorStyle.Render(author),
				commentStyle.Render(commentText),
				commentMarker.Render(commentIndicator),
				"\n",
			}, separator)
	}

	idFormatFunc := func() string {
		return idStyle.Render(idText) + "\n"
	}

	compactFormatFunc := func() string {
		return strings.Join(
			[]string{
				idStyle.Render(idText),
				statusStyle.Render(statusText),
				titleStyle.Render(title),
				authorStyle.Render(author),
				"\n",
			}, separator)
	}

	var (
		_ formatFunc = defaultFormatFunc
		_ formatFunc = idFormatFunc
		_ formatFunc = compactFormatFunc
	)

	var format string

	flag.StringVar(&format, "format", "default", "")
	flag.Parse()

	fn := defaultFormatFunc

	switch format {
	case "compact":
		fn = compactFormatFunc
	case "id":
		fn = idFormatFunc
	}

	os.Stderr.Write([]byte(fn()))
}
