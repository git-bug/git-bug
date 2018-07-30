package termui

import (
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/repository"
	"github.com/jroimartin/gocui"
)

type termUI struct {
	cache    cache.RepoCacher
	bugTable *bugTable
}

var ui *termUI

func Run(repo repository.Repo) error {
	c := cache.NewRepoCache(repo)

	ui = &termUI{
		cache:    c,
		bugTable: newBugTable(c),
	}

	g, err := gocui.NewGui(gocui.OutputNormal)

	if err != nil {
		return err
	}

	defer g.Close()

	g.SetManagerFunc(layout)

	err = keybindings(g)

	if err != nil {
		return err
	}

	err = g.MainLoop()

	if err != nil && err != gocui.ErrQuit {
		return err
	}

	return nil
}

func layout(g *gocui.Gui) error {
	//maxX, maxY := g.Size()

	ui.bugTable.layout(g)

	v, err := g.View("table")
	if err != nil {
		return err
	}

	cursorClamp(v)

	return nil
}

func keybindings(g *gocui.Gui) error {
	if err := g.SetKeybinding("", 'q', gocui.ModNone, quit); err != nil {
		return err
	}
	if err := g.SetKeybinding("table", 'j', gocui.ModNone, cursorDown); err != nil {
		return err
	}
	if err := g.SetKeybinding("table", gocui.KeyArrowDown, gocui.ModNone, cursorDown); err != nil {
		return err
	}
	if err := g.SetKeybinding("table", 'k', gocui.ModNone, cursorUp); err != nil {
		return err
	}
	if err := g.SetKeybinding("table", gocui.KeyArrowUp, gocui.ModNone, cursorUp); err != nil {
		return err
	}

	//err = g.SetKeybinding("table", 'h', gocui.ModNone, popTable)
	//err = g.SetKeybinding("table", gocui.KeyArrowLeft, gocui.ModNone, popTable)
	//err = g.SetKeybinding("table", 'l', gocui.ModNone, pushTable)
	//err = g.SetKeybinding("table", gocui.KeyArrowRight, gocui.ModNone, pushTable)
	//err = g.SetKeybinding("table", 'p', gocui.ModNone, playSelected)
	//err = g.SetKeybinding("table", gocui.KeyEnter, gocui.ModNone, playSelectedAndExit)
	//err = g.SetKeybinding("table", 'm', gocui.ModNone, loadNextRecords)

	return nil
}

func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}

func cursorDown(g *gocui.Gui, v *gocui.View) error {
	_, y := v.Cursor()
	y = minInt(y+1, ui.bugTable.getTableLength()-1)

	err := v.SetCursor(0, y)
	if err != nil {
		return err
	}

	return nil
}

func cursorUp(g *gocui.Gui, v *gocui.View) error {
	_, y := v.Cursor()
	y = maxInt(y-1, 0)

	err := v.SetCursor(0, y)
	if err != nil {
		return err
	}

	return nil
}

func cursorClamp(v *gocui.View) error {
	_, y := v.Cursor()

	y = minInt(y, ui.bugTable.getTableLength()-1)
	y = maxInt(y, 0)

	err := v.SetCursor(0, y)
	if err != nil {
		return err
	}

	return nil
}
