package termui

import (
	"fmt"

	"github.com/MichaelMure/git-bug/util"
	"github.com/jroimartin/gocui"
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
	wrapped, nblines := util.WordWrap(ep.message, width-2)
	height := minInt(nblines+1, maxY)
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

func (ep *errorPopup) Activate(message string) {
	ep.message = message
}
