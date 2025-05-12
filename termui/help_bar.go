package termui

import (
	"fmt"
	"strings"

	text "github.com/MichaelMure/go-term-text"

	"github.com/git-bug/git-bug/util/colors"
)

type helpBar []struct {
	keys string
	text string
}

func (hb helpBar) Render(maxX int) string {
	var builder strings.Builder

	il := len(hb) - 1
	for i, entry := range hb {
		builder.WriteString(colors.White(colors.BlackBg(fmt.Sprintf("[%s] %s", entry.keys, entry.text))))

		if i < il {
			builder.WriteString("  ")
		}
	}

	tl := text.Len(builder.String())
	if tl < maxX {
		builder.WriteString(colors.White(colors.BlackBg(strings.Repeat(" ", maxX-tl))))
	}

	return builder.String()
}
