package bug

import (
	"crypto/sha1"
	"fmt"
	"io"
	"strings"

	"github.com/MichaelMure/git-bug/util/text"
)

// RGBColor is a color type in the form red, green, blue
type RGBColor struct {
	red   uint8
	green uint8
	blue  uint8
}

type Label string

func (l Label) String() string {
	return string(l)
}

// RGBColor from a Label computed in a deterministic way
func (l Label) RGBColor() RGBColor {
	id := 0
	hash := sha1.Sum([]byte(l))

	// colors from: https://material-ui.com/style/color/
	colors := []RGBColor{
		RGBColor{red: 244, green: 67, blue: 54},   // red
		RGBColor{red: 233, green: 30, blue: 99},   // pink
		RGBColor{red: 156, green: 39, blue: 176},  // purple
		RGBColor{red: 103, green: 58, blue: 183},  // deepPurple
		RGBColor{red: 63, green: 81, blue: 181},   // indigo
		RGBColor{red: 33, green: 150, blue: 243},  // blue
		RGBColor{red: 3, green: 169, blue: 244},   // lightBlue
		RGBColor{red: 0, green: 188, blue: 212},   // cyan
		RGBColor{red: 0, green: 150, blue: 136},   // teal
		RGBColor{red: 76, green: 175, blue: 80},   // green
		RGBColor{red: 139, green: 195, blue: 74},  // lightGreen
		RGBColor{red: 205, green: 220, blue: 57},  // lime
		RGBColor{red: 255, green: 235, blue: 59},  // yellow
		RGBColor{red: 255, green: 193, blue: 7},   // amber
		RGBColor{red: 255, green: 152, blue: 0},   // orange
		RGBColor{red: 255, green: 87, blue: 34},   // deepOrange
		RGBColor{red: 121, green: 85, blue: 72},   // brown
		RGBColor{red: 158, green: 158, blue: 158}, // grey
		RGBColor{red: 96, green: 125, blue: 139},  // blueGrey
	}

	for _, char := range hash {
		id = (id + int(char)) % len(colors)
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
