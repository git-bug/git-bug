package termui

import (
	"fmt"
	"strings"

	text "github.com/MichaelMure/go-term-text"

	"github.com/MichaelMure/git-bug/util/colors"
)

type helpBar []struct {
	keys string
	text string
}

func (hb helpBar) Render(maxX int) string {
	var builder strings.Builder
	for _, entry := range hb {
		builder.WriteString(colors.White(colors.BlueBg(fmt.Sprintf("[%s] %s", entry.keys, entry.text))))
		builder.WriteByte(' ')
	}

	l := text.Len(builder.String())
	if l < maxX {
		builder.WriteString(colors.White(colors.BlueBg(strings.Repeat(" ", maxX-l))))
	}

	return builder.String()
}
