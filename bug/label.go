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

// RGBA from a Label computed in a deterministic way
func (l Label) RGBA() color.RGBA {
	id := 0
	hash := sha1.Sum([]byte(l))

	// colors from: https://material-ui.com/style/color/
	colors := []color.RGBA{
		color.RGBA{R: 244, G: 67, B: 54, A: 255},   // red
		color.RGBA{R: 233, G: 30, B: 99, A: 255},   // pink
		color.RGBA{R: 156, G: 39, B: 176, A: 255},  // purple
		color.RGBA{R: 103, G: 58, B: 183, A: 255},  // deepPurple
		color.RGBA{R: 63, G: 81, B: 181, A: 255},   // indigo
		color.RGBA{R: 33, G: 150, B: 243, A: 255},  // blue
		color.RGBA{R: 3, G: 169, B: 244, A: 255},   // lightBlue
		color.RGBA{R: 0, G: 188, B: 212, A: 255},   // cyan
		color.RGBA{R: 0, G: 150, B: 136, A: 255},   // teal
		color.RGBA{R: 76, G: 175, B: 80, A: 255},   // green
		color.RGBA{R: 139, G: 195, B: 74, A: 255},  // lightGreen
		color.RGBA{R: 205, G: 220, B: 57, A: 255},  // lime
		color.RGBA{R: 255, G: 235, B: 59, A: 255},  // yellow
		color.RGBA{R: 255, G: 193, B: 7, A: 255},   // amber
		color.RGBA{R: 255, G: 152, B: 0, A: 255},   // orange
		color.RGBA{R: 255, G: 87, B: 34, A: 255},   // deepOrange
		color.RGBA{R: 121, G: 85, B: 72, A: 255},   // brown
		color.RGBA{R: 158, G: 158, B: 158, A: 255}, // grey
		color.RGBA{R: 96, G: 125, B: 139, A: 255},  // blueGrey
	}

	for _, char := range hash {
		id = (id + int(char)) % len(colors)
	}

	return colors[id]
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
