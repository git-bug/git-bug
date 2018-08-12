package termui

import (
	"fmt"

	"github.com/MichaelMure/git-bug/util"
	"github.com/jroimartin/gocui"
)

const msgPopupView = "msgPopupView"

const msgPopupErrorTitle = "Error"

type msgPopup struct {
	active  bool
	title   string
	message string
}

func newMsgPopup() *msgPopup {
	return &msgPopup{
		message: "",
	}
}

func (ep *msgPopup) keybindings(g *gocui.Gui) error {
	if err := g.SetKeybinding(msgPopupView, gocui.KeySpace, gocui.ModNone, ep.close); err != nil {
		return err
	}
	if err := g.SetKeybinding(msgPopupView, gocui.KeyEnter, gocui.ModNone, ep.close); err != nil {
		return err
	}
	if err := g.SetKeybinding(msgPopupView, 'q', gocui.ModNone, ep.close); err != nil {
		return err
	}

	return nil
}

func (ep *msgPopup) layout(g *gocui.Gui) error {
	if !ep.active {
		return nil
	}

	maxX, maxY := g.Size()

	width := minInt(60, maxX)
	wrapped, lines := util.TextWrap(ep.message, width-2)
	height := minInt(lines+1, maxY-3)
	x0 := (maxX - width) / 2
	y0 := (maxY - height) / 2

	v, err := g.SetView(msgPopupView, x0, y0, x0+width, y0+height)
	if err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}

		v.Frame = true
		v.Autoscroll = true
	}

	v.Title = ep.title

	v.Clear()
	fmt.Fprintf(v, wrapped)

	if _, err := g.SetCurrentView(msgPopupView); err != nil {
		return err
	}

	return nil
}

func (ep *msgPopup) close(g *gocui.Gui, v *gocui.View) error {
	ep.active = false
	ep.message = ""
	return g.DeleteView(msgPopupView)
}

func (ep *msgPopup) Activate(title string, message string) {
	ep.active = true
	ep.title = title
	ep.message = message
}

func (ep *msgPopup) UpdateMessage(message string) {
	ep.message = message
}
