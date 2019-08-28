package bug

import (
	"crypto/sha1"
	"fmt"
	"image/color"
	"strings"

	"github.com/MichaelMure/git-bug/util/text"
)

type Label string

func (l Label) String() string {
	return string(l)
}

type LabelColor color.RGBA

// RGBA from a Label computed in a deterministic way
func (l Label) Color() LabelColor {
	id := 0
	hash := sha1.Sum([]byte(l))

	// colors from: https://material-ui.com/style/color/
	colors := []LabelColor{
		LabelColor{R: 244, G: 67, B: 54, A: 255},   // red
		LabelColor{R: 233, G: 30, B: 99, A: 255},   // pink
		LabelColor{R: 156, G: 39, B: 176, A: 255},  // purple
		LabelColor{R: 103, G: 58, B: 183, A: 255},  // deepPurple
		LabelColor{R: 63, G: 81, B: 181, A: 255},   // indigo
		LabelColor{R: 33, G: 150, B: 243, A: 255},  // blue
		LabelColor{R: 3, G: 169, B: 244, A: 255},   // lightBlue
		LabelColor{R: 0, G: 188, B: 212, A: 255},   // cyan
		LabelColor{R: 0, G: 150, B: 136, A: 255},   // teal
		LabelColor{R: 76, G: 175, B: 80, A: 255},   // green
		LabelColor{R: 139, G: 195, B: 74, A: 255},  // lightGreen
		LabelColor{R: 205, G: 220, B: 57, A: 255},  // lime
		LabelColor{R: 255, G: 235, B: 59, A: 255},  // yellow
		LabelColor{R: 255, G: 193, B: 7, A: 255},   // amber
		LabelColor{R: 255, G: 152, B: 0, A: 255},   // orange
		LabelColor{R: 255, G: 87, B: 34, A: 255},   // deepOrange
		LabelColor{R: 121, G: 85, B: 72, A: 255},   // brown
		LabelColor{R: 158, G: 158, B: 158, A: 255}, // grey
		LabelColor{R: 96, G: 125, B: 139, A: 255},  // blueGrey
	}

	for _, char := range hash {
		id = (id + int(char)) % len(colors)
	}

	return colors[id]
}

func (lc LabelColor) RGBA() color.RGBA {
	return color.RGBA(lc)
}

type Term256 int

func (lc LabelColor) Term256() Term256 {
	red := Term256(lc.R) * 6 / 256
	green := Term256(lc.G) * 6 / 256
	blue := Term256(lc.B) * 6 / 256

	return red*36 + green*6 + blue + 16
}

func (t Term256) Escape() string {
	return fmt.Sprintf("\x1b[38;5;%dm", t)
}

func (t Term256) Unescape() string {
	return "\x1b[0m"
}

func (l Label) Validate() error {
	str := string(l)

	if text.Empty(str) {
		return fmt.Errorf("empty")
	}

	if strings.Contains(str, "\n") {
		return fmt.Errorf("should be a single line")
	}

	if !text.Safe(str) {
		return fmt.Errorf("not fully printable")
	}

	return nil
}
