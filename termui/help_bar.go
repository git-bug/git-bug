package termui

import (
	"fmt"
	"strings"

	"github.com/MichaelMure/git-bug/util/colors"
)

type helpBar []struct {
	keys string
	text string
}

func (hb helpBar) Render() string {
	var builder strings.Builder
	for i, entry := range hb {
		if i != 0 {
			builder.WriteByte(' ')
		}
		builder.WriteString(colors.BlueBg(fmt.Sprintf("[%s] %s", entry.keys, entry.text)))
	}
	return builder.String()
}
