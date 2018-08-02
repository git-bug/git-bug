package termui

import (
	"fmt"
	"github.com/jroimartin/gocui"
	"strings"
)

const errorPopupView = "errorPopupView"

type errorPopup struct {
	message string
}

func newErrorPopup() *errorPopup {
	return &errorPopup{
		message: "",
	}
}

func (ep *errorPopup) keybindings(g *gocui.Gui) error {
	if err := g.SetKeybinding(errorPopupView, gocui.KeySpace, gocui.ModNone, ep.close); err != nil {
		return err
	}
	if err := g.SetKeybinding(errorPopupView, gocui.KeyEnter, gocui.ModNone, ep.close); err != nil {
		return err
	}

	return nil
}

func (ep *errorPopup) layout(g *gocui.Gui) error {
	if ep.message == "" {
		return nil
	}

	maxX, maxY := g.Size()

	width := minInt(30, maxX)
	wrapped, nblines := word_wrap(ep.message, width-2)
	height := minInt(nblines+2, maxY)
	x0 := (maxX - width) / 2
	y0 := (maxY - height) / 2

	v, err := g.SetView(errorPopupView, x0, y0, x0+width, y0+height)
	if err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}

		v.Frame = true
		v.Title = "Error"

		fmt.Fprintf(v, wrapped)
	}

	if _, err := g.SetCurrentView(errorPopupView); err != nil {
		return err
	}

	return nil
}

func (ep *errorPopup) close(g *gocui.Gui, v *gocui.View) error {
	ep.message = ""
	return g.DeleteView(errorPopupView)
}

func (ep *errorPopup) activate(message string) {
	ep.message = message
}

func word_wrap(text string, lineWidth int) (string, int) {
	words := strings.Fields(strings.TrimSpace(text))
	if len(words) == 0 {
		return text, 1
	}
	lines := 1
	wrapped := words[0]
	spaceLeft := lineWidth - len(wrapped)
	for _, word := range words[1:] {
		if len(word)+1 > spaceLeft {
			wrapped += "\n" + word
			spaceLeft = lineWidth - len(word)
			lines++
		} else {
			wrapped += " " + word
			spaceLeft -= 1 + len(word)
		}
	}

	return wrapped, lines
}
