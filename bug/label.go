package bug

import (
	"fmt"
	"io"
	"strings"

	"github.com/MichaelMure/git-bug/util/text"
)

// Color in the form red, green, blue
type Color struct {
	red   uint8
	green uint8
	blue  uint8
}

type Label string

func (l Label) String() string {
	return string(l)
}

// Color from a Label in a deterministic way
func (l Label) Color() Color {
	label := string(l)
	id := 0

	// colors from: https://material-ui.com/style/color/
	colors := []Color{
		Color{red: 244, green: 67, blue: 54},   // red
		Color{red: 233, green: 30, blue: 99},   // pink
		Color{red: 156, green: 39, blue: 176},  // purple
		Color{red: 103, green: 58, blue: 183},  // deepPurple
		Color{red: 63, green: 81, blue: 181},   // indigo
		Color{red: 33, green: 150, blue: 243},  // blue
		Color{red: 3, green: 169, blue: 244},   // lightBlue
		Color{red: 0, green: 188, blue: 212},   // cyan
		Color{red: 0, green: 150, blue: 136},   // teal
		Color{red: 76, green: 175, blue: 80},   // green
		Color{red: 139, green: 195, blue: 74},  // lightGreen
		Color{red: 205, green: 220, blue: 57},  // lime
		Color{red: 255, green: 235, blue: 59},  // yellow
		Color{red: 255, green: 193, blue: 7},   // amber
		Color{red: 255, green: 152, blue: 0},   // orange
		Color{red: 255, green: 87, blue: 34},   // deepOrange
		Color{red: 121, green: 85, blue: 72},   // brown
		Color{red: 158, green: 158, blue: 158}, // grey
		Color{red: 96, green: 125, blue: 139},  // blueGrey
	}

	for pos, char := range label {
		id = ((pos+1)*(id+1) + int(char)) % len(colors)
	}

	return colors[id]
}

// UnmarshalGQL implements the graphql.Unmarshaler interface
func (l *Label) UnmarshalGQL(v interface{}) error {
	_, ok := v.(string)
	if !ok {
		return fmt.Errorf("labels must be strings")
	}

	*l = v.(Label)

	return nil
}

// MarshalGQL implements the graphql.Marshaler interface
func (l Label) MarshalGQL(w io.Writer) {
	_, _ = w.Write([]byte(`"` + l.String() + `"`))
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
