package text

import (
	"fmt"
	"strconv"
	"strings"
)

const Escape = '\x1b'

type EscapeState struct {
	Bold       bool
	Dim        bool
	Italic     bool
	Underlined bool
	Blink      bool
	Reverse    bool
	Hidden     bool
	CrossedOut bool

	FgColor Color
	BgColor Color
}

type Color interface {
	Codes() []string
}

func (es *EscapeState) Witness(s string) {
	inEscape := false
	var start int

	runes := []rune(s)

	for i, r := range runes {
		if r == Escape {
			inEscape = true
			start = i
			continue
		}
		if inEscape {
			if r == 'm' {
				inEscape = false
				es.witnessCode(string(runes[start+1 : i]))
			}
			continue
		}
	}
}

func (es *EscapeState) witnessCode(s string) {
	if s == "" {
		return
	}
	if s == "[" {
		es.reset()
		return
	}
	if len(s) < 2 {
		return
	}
	if s[0] != '[' {
		return
	}

	s = s[1:]
	split := strings.Split(s, ";")

	dequeue := func() {
		split = split[1:]
	}

	color := func(ground int) Color {
		if len(split) < 1 {
			// the whole sequence is broken, ignoring the rest
			return nil
		}

		subCode := split[0]
		dequeue()

		switch subCode {
		case "2":
			if len(split) < 3 {
				return nil
			}
			r, err := strconv.Atoi(split[0])
			dequeue()
			if err != nil {
				return nil
			}
			g, err := strconv.Atoi(split[0])
			dequeue()
			if err != nil {
				return nil
			}
			b, err := strconv.Atoi(split[0])
			dequeue()
			if err != nil {
				return nil
			}
			return &ColorRGB{ground: ground, R: r, G: g, B: b}

		case "5":
			if len(split) < 1 {
				return nil
			}
			index, err := strconv.Atoi(split[0])
			dequeue()
			if err != nil {
				return nil
			}
			return &Color256{ground: ground, Index: index}

		}
		return nil
	}

	for len(split) > 0 {
		code, err := strconv.Atoi(split[0])
		if err != nil {
			return
		}
		dequeue()

		switch {
		case code == 0:
			es.reset()

		case code == 1:
			es.Bold = true
		case code == 2:
			es.Dim = true
		case code == 3:
			es.Italic = true
		case code == 4:
			es.Underlined = true
		case code == 5:
			es.Blink = true
		// case code == 6:
		case code == 7:
			es.Reverse = true
		case code == 8:
			es.Hidden = true
		case code == 9:
			es.CrossedOut = true

		case code == 21:
			es.Bold = false
		case code == 22:
			es.Dim = false
		case code == 23:
			es.Italic = false
		case code == 24:
			es.Underlined = false
		case code == 25:
			es.Blink = false
		// case code == 26:
		case code == 27:
			es.Reverse = false
		case code == 28:
			es.Hidden = false
		case code == 29:
			es.CrossedOut = false

		case (code >= 30 && code <= 37) || code == 39 || (code >= 90 && code <= 97):
			es.FgColor = ColorIndex(code)

		case (code >= 40 && code <= 47) || code == 49 || (code >= 100 && code <= 107):
			es.BgColor = ColorIndex(code)

		case code == 38:
			es.FgColor = color(code)
			if es.FgColor == nil {
				return
			}

		case code == 48:
			es.BgColor = color(code)
			if es.BgColor == nil {
				return
			}
		}
	}
}

func (es *EscapeState) reset() {
	*es = EscapeState{}
}

func (es *EscapeState) String() string {
	var codes []string

	if es.Bold {
		codes = append(codes, strconv.Itoa(1))
	}
	if es.Dim {
		codes = append(codes, strconv.Itoa(2))
	}
	if es.Italic {
		codes = append(codes, strconv.Itoa(3))
	}
	if es.Underlined {
		codes = append(codes, strconv.Itoa(4))
	}
	if es.Blink {
		codes = append(codes, strconv.Itoa(5))
	}
	if es.Reverse {
		codes = append(codes, strconv.Itoa(7))
	}
	if es.Hidden {
		codes = append(codes, strconv.Itoa(8))
	}
	if es.CrossedOut {
		codes = append(codes, strconv.Itoa(9))
	}

	if es.FgColor != nil {
		codes = append(codes, es.FgColor.Codes()...)
	}
	if es.BgColor != nil {
		codes = append(codes, es.BgColor.Codes()...)
	}

	if len(codes) == 0 {
		return "\x1b[0m"
	}

	return fmt.Sprintf("\x1b[%sm", strings.Join(codes, ";"))
}

type ColorIndex int

func (cInd ColorIndex) Codes() []string {
	return []string{strconv.Itoa(int(cInd))}
}

type Color256 struct {
	ground int
	Index  int
}

func (c256 Color256) Codes() []string {
	return []string{
		strconv.Itoa(c256.ground),
		"5",
		strconv.Itoa(c256.Index),
	}
}

type ColorRGB struct {
	ground  int
	R, G, B int
}

func (cRGB ColorRGB) Codes() []string {
	return []string{
		strconv.Itoa(cRGB.ground),
		"2",
		strconv.Itoa(cRGB.R),
		strconv.Itoa(cRGB.G),
		strconv.Itoa(cRGB.B),
	}
}
