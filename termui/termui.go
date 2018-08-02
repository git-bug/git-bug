package termui

import (
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/input"
	"github.com/MichaelMure/git-bug/repository"
	"github.com/jroimartin/gocui"
	"github.com/pkg/errors"
)

var errTerminateMainloop = errors.New("terminate gocui mainloop")

type termUI struct {
	g      *gocui.Gui
	gError chan error
	cache  cache.RepoCacher

	activeWindow window

	bugTable   *bugTable
	showBug    *showBug
	errorPopup *errorPopup
}

func (tui *termUI) activateWindow(window window) error {
	if err := tui.activeWindow.disable(tui.g); err != nil {
		return err
	}

	tui.activeWindow = window

	return nil
}

var ui *termUI

type window interface {
	keybindings(g *gocui.Gui) error
	layout(g *gocui.Gui) error
	disable(g *gocui.Gui) error
}

func Run(repo repository.Repo) error {
	c := cache.NewRepoCache(repo)

	ui = &termUI{
		gError:     make(chan error, 1),
		cache:      c,
		bugTable:   newBugTable(c),
		showBug:    newShowBug(c),
		errorPopup: newErrorPopup(),
	}

	ui.activeWindow = ui.bugTable

	initGui()

	err := <-ui.gError

	if err != nil && err != gocui.ErrQuit {
		return err
	}

	return nil
}

func initGui() {
	g, err := gocui.NewGui(gocui.OutputNormal)

	if err != nil {
		ui.gError <- err
		return
	}

	ui.g = g

	ui.g.SetManagerFunc(layout)

	err = keybindings(ui.g)

	if err != nil {
		ui.g.Close()
		ui.g = nil
		ui.gError <- err
		return
	}

	err = g.MainLoop()

	if err != nil && err != errTerminateMainloop {
		if ui.g != nil {
			ui.g.Close()
		}
		ui.gError <- err
	}

	return
}

func layout(g *gocui.Gui) error {
	g.Cursor = false

	if err := ui.activeWindow.layout(g); err != nil {
		return err
	}

	if err := ui.errorPopup.layout(g); err != nil {
		return err
	}

	return nil
}

func keybindings(g *gocui.Gui) error {
	// Quit
	if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		return err
	}

	if err := ui.bugTable.keybindings(g); err != nil {
		return err
	}

	if err := ui.showBug.keybindings(g); err != nil {
		return err
	}

	if err := ui.errorPopup.keybindings(g); err != nil {
		return err
	}

	return nil
}

func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}

func newBugWithEditor(g *gocui.Gui, v *gocui.View) error {
	// This is somewhat hacky.
	// As there is no way to pause gocui, run the editor, restart gocui,
	// we have to stop it entirely and start a new one later.
	//
	// - an error channel is used to route the returned error of this new
	// 		instance into the original launch function
	// - a custom error (errTerminateMainloop) is used to terminate the original
	//		instance's mainLoop. This error is then filtered.

	ui.g.Close()
	ui.g = nil

	title, message, err := input.BugCreateEditorInput(ui.cache.Repository(), "", "")

	if err != nil && err != input.ErrEmptyTitle {
		return err
	}

	if err == input.ErrEmptyTitle {
		ui.errorPopup.activate("Empty title, aborting.")
	} else {
		_, err = ui.cache.NewBug(title, message)
		if err != nil {
			return err
		}
	}

	initGui()

	return errTerminateMainloop
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a > b {
		return b
	}
	return a
}
