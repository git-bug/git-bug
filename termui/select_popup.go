package termui

import (
	"fmt"
	"strings"

	"github.com/jroimartin/gocui"
)

const selectPopupView = "selectPopupView"
const showSelectInstructions = "showSelectInstructions"

type selectPopup struct {
	active       bool
	title        string
	options      []string
	optionSelect []bool
	selected     int
	childViews   []string
	c            chan []string
}

func newSelectPopup() *selectPopup {
	return &selectPopup{}
}

func (sp *selectPopup) keybindings(g *gocui.Gui) error {
	// Close
	if err := g.SetKeybinding(selectPopupView, gocui.KeyEsc, gocui.ModNone, sp.close); err != nil {
		return err
	}
	if err := g.SetKeybinding(selectPopupView, 'q', gocui.ModNone, sp.close); err != nil {
		return err
	}

	// Validate
	if err := g.SetKeybinding(selectPopupView, gocui.KeyEnter, gocui.ModNone, sp.validate); err != nil {
		return err
	}

	// Up
	if err := g.SetKeybinding(selectPopupView, gocui.KeyArrowUp, gocui.ModNone, sp.selectPrevious); err != nil {
		return err
	}
	if err := g.SetKeybinding(selectPopupView, 'k', gocui.ModNone, sp.selectPrevious); err != nil {
		return err
	}

	// Down
	if err := g.SetKeybinding(selectPopupView, gocui.KeyArrowDown, gocui.ModNone, sp.selectNext); err != nil {
		return err
	}
	if err := g.SetKeybinding(selectPopupView, 'j', gocui.ModNone, sp.selectNext); err != nil {
		return err
	}

	// Select
	if err := g.SetKeybinding(selectPopupView, gocui.KeySpace, gocui.ModNone, sp.selectItem); err != nil {
		return err
	}
	if err := g.SetKeybinding(selectPopupView, 'x', gocui.ModNone, sp.selectItem); err != nil {
		return err
	}

	// Add
	if err := g.SetKeybinding(selectPopupView, 'a', gocui.ModNone, sp.addItem); err != nil {
		return err
	}

	return nil
}

func (sp *selectPopup) layout(g *gocui.Gui) error {
	if !sp.active {
		return nil
	}

	maxX, maxY := g.Size()

	// TODO: Make width adaptive
	width := 30
	height := 2*len(sp.options) + 3
	x0 := (maxX - width) / 2
	y0 := (maxY - height) / 2

	v, err := g.SetView(selectPopupView, x0, y0, x0+width, y0+height)
	if err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}

		v.Frame = true
		v.Title = sp.title
	}

	y0 += 1

	for i, name := range sp.options {
		viewname := fmt.Sprintf("view%d", i)
		v, err = g.SetView(viewname, x0+2, y0, x0+width-2, y0+2)
		sp.childViews = append(sp.childViews, viewname)
		v.Frame = i == sp.selected
		v.Clear()

		selectBox := " [ ] "
		if sp.optionSelect[i] {
			selectBox = " [x] "
		}

		fmt.Fprint(v, selectBox + name)

		y0 += 2
	}

	v, err = g.SetView(showSelectInstructions, x0, y0, x0+width, y0+2)
	sp.childViews = append(sp.childViews, showSelectInstructions)
	if err != nil {
		if err != gocui.ErrUnknownView{
			return err
		}

		v.Frame = false
		v.BgColor = gocui.ColorBlue
	}

	v.Clear()
	fmt.Fprint(v, "[↓↑,jk] Nav [a] Add item")

	if _, err = g.SetViewOnTop(showSelectInstructions); err != nil {
		return err
	}

	if _, err := g.SetCurrentView(selectPopupView); err != nil {
		return err
	}

	return nil
}

func(sp *selectPopup) selectPrevious(g *gocui.Gui, v*gocui.View) error {
	sp.selected = maxInt(0, sp.selected-1)

	return nil
}

func(sp *selectPopup) selectNext(g *gocui.Gui, v*gocui.View) error {
	sp.selected = minInt(len(sp.options)-1, sp.selected+1)

	return nil
}

func(sp *selectPopup) selectItem(g *gocui.Gui, v*gocui.View) error {
	sp.optionSelect[sp.selected] = !sp.optionSelect[sp.selected]

	return nil
}

func (sp *selectPopup) addItem(g *gocui.Gui, v *gocui.View) error {
	c := ui.inputPopup.Activate("Add item")

	go func() {
		input := <-c
		input = strings.TrimSuffix(input, "\n")
		input = strings.Replace(input, " ", "-", -1)

		sp.options = append(sp.options, input)
		sp.optionSelect = append(sp.optionSelect, true)
	}()

	return nil
}

func (sp *selectPopup) close(g *gocui.Gui, v *gocui.View) error {
	sp.title = ""
	sp.active = false

	for _, v := range sp.childViews {
		if err := g.DeleteView(v); err != nil && err != gocui.ErrUnknownView {
			return err
		}
	}

	return g.DeleteView(selectPopupView)
}

func (sp *selectPopup) validate(g *gocui.Gui, v *gocui.View) error {
	sp.title = ""

	if err := sp.close(g, v); err != nil {
		return err
	}

	selectedOptions := []string{}
	for i, option := range sp.options {
		if sp.optionSelect[i] {
			selectedOptions = append(selectedOptions, option)
		}
	}

	sp.c <- selectedOptions

	return nil
}

func (sp *selectPopup) Activate(title string, options []string, sel bool) <-chan []string {
	sp.title = title
	sp.options = options
	sp.optionSelect = make([]bool, len(options))
	for i, _ := range sp.optionSelect {
		sp.optionSelect[i] = sel
	}
	sp.selected = 0
	sp.active = true
	sp.c = make(chan []string)

	return sp.c
}