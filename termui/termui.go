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

	v, err := g.View("bugTable")
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
	if err := g.SetKeybinding("bugTable", 'j', gocui.ModNone, cursorDown); err != nil {
		return err
	}
	if err := g.SetKeybinding("bugTable", gocui.KeyArrowDown, gocui.ModNone, cursorDown); err != nil {
		return err
	}
	if err := g.SetKeybinding("bugTable", 'k', gocui.ModNone, cursorUp); err != nil {
		return err
	}
	if err := g.SetKeybinding("bugTable", gocui.KeyArrowUp, gocui.ModNone, cursorUp); err != nil {
		return err
	}

	if err := g.SetKeybinding("bugTable", 'h', gocui.ModNone, previousPage); err != nil {
		return err
	}
	if err := g.SetKeybinding("bugTable", gocui.KeyArrowLeft, gocui.ModNone, previousPage); err != nil {
		return err
	}
	if err := g.SetKeybinding("bugTable", gocui.KeyPgup, gocui.ModNone, previousPage); err != nil {
		return err
	}
	if err := g.SetKeybinding("bugTable", 'l', gocui.ModNone, nextPage); err != nil {
		return err
	}
	if err := g.SetKeybinding("bugTable", gocui.KeyArrowRight, gocui.ModNone, nextPage); err != nil {
		return err
	}
	if err := g.SetKeybinding("bugTable", gocui.KeyPgup, gocui.ModNone, nextPage); err != nil {
		return err
	}

	//err = g.SetKeybinding("bugTable", 'p', gocui.ModNone, playSelected)
	//err = g.SetKeybinding("bugTable", gocui.KeyEnter, gocui.ModNone, playSelectedAndExit)
	//err = g.SetKeybinding("bugTable", 'm', gocui.ModNone, loadNextRecords)

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

func nextPage(g *gocui.Gui, v *gocui.View) error {
	_, maxY := v.Size()
	return ui.bugTable.nextPage(maxY)
}

func previousPage(g *gocui.Gui, v *gocui.View) error {
	_, maxY := v.Size()
	return ui.bugTable.previousPage(maxY)
}
