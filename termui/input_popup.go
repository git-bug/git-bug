package termui

import (
	"errors"
	"io/ioutil"

	"github.com/awesome-gocui/gocui"
)

const inputPopupView = "inputPopupView"

// inputPopup is a simple popup with an input field
type inputPopup struct {
	active  bool
	title   string
	preload string
	c       chan string
}

func newInputPopup() *inputPopup {
	return &inputPopup{}
}

func (ip *inputPopup) keybindings(g *gocui.Gui) error {
	// Close
	if err := g.SetKeybinding(inputPopupView, gocui.KeyEsc, gocui.ModNone, ip.close); err != nil {
		return err
	}

	// Validate
	if err := g.SetKeybinding(inputPopupView, gocui.KeyEnter, gocui.ModNone, ip.validate); err != nil {
		return err
	}

	return nil
}

func (ip *inputPopup) layout(g *gocui.Gui) error {
	if !ip.active {
		return nil
	}

	maxX, maxY := g.Size()

	width := minInt(30, maxX)
	height := 2
	x0 := (maxX - width) / 2
	y0 := (maxY - height) / 2

	v, err := g.SetView(inputPopupView, x0, y0, x0+width, y0+height, 0)
	if err != nil {
		if !errors.Is(err, gocui.ErrUnknownView) {
			return err
		}

		v.Frame = true
		v.Title = ip.title
		v.Editable = true
		_, err = v.Write([]byte(ip.preload))
		if err != nil {
			return err
		}
	}

	if _, err := g.SetCurrentView(inputPopupView); err != nil {
		return err
	}

	return nil
}

func (ip *inputPopup) close(g *gocui.Gui, v *gocui.View) error {
	ip.title = ""
	ip.active = false
	return g.DeleteView(inputPopupView)
}

func (ip *inputPopup) validate(g *gocui.Gui, v *gocui.View) error {
	ip.title = ""

	content, err := ioutil.ReadAll(v)
	if err != nil {
		return err
	}

	ip.title = ""
	ip.active = false
	err = g.DeleteView(inputPopupView)
	if err != nil {
		return err
	}

	ip.c <- string(content)

	return nil
}

func (ip *inputPopup) ActivateWithContent(title string, content string) <-chan string {
	ip.preload = content
	return ip.Activate(title)
}

func (ip *inputPopup) Activate(title string) <-chan string {
	ip.title = title
	ip.active = true
	ip.c = make(chan string)
	return ip.c
}
