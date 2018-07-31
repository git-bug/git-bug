package termui

import (
	"fmt"
	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/util"
	"github.com/jroimartin/gocui"
)

const bugTableView = "bugTableView"

type bugTable struct {
	cache  cache.RepoCacher
	allIds []string
	bugs   []*bug.Snapshot
	cursor int
}

func newBugTable(cache cache.RepoCacher) *bugTable {
	return &bugTable{
		cache:  cache,
		cursor: 0,
	}
}

func (bt *bugTable) layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()

	v, err := g.SetView("header", -1, -1, maxX, 3)

	if err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}

		v.Frame = false
	}

	v.Clear()
	bt.renderHeader(v, maxX)

	v, err = g.SetView(bugTableView, -1, 1, maxX, maxY-2)

	if err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}

		v.Frame = false
		v.Highlight = true
		v.SelBgColor = gocui.ColorWhite
		v.SelFgColor = gocui.ColorBlack

		_, err = g.SetCurrentView(bugTableView)

		if err != nil {
			return err
		}
	}

	_, viewHeight := v.Size()
	err = bt.paginate(viewHeight - 1)
	if err != nil {
		return err
	}

	err = bt.cursorClamp(v)
	if err != nil {
		return err
	}

	v.Clear()
	bt.render(v, maxX)

	v, err = g.SetView("footer", -1, maxY-3, maxX, maxY)

	if err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}

		v.Frame = false
	}

	v.Clear()
	bt.renderFooter(v, maxX)

	v, err = g.SetView("instructions", -1, maxY-2, maxX, maxY)

	if err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}

		v.Frame = false
		v.BgColor = gocui.ColorBlue

		fmt.Fprintf(v, "[q] Quit [←,h] Previous page [↓,j] Down [↑,k] Up [→,l] Next page [enter] Open bug [n] New bug")
	}

	return nil
}

func (bt *bugTable) keybindings(g *gocui.Gui) error {
	// Quit
	if err := g.SetKeybinding(bugTableView, 'q', gocui.ModNone, quit); err != nil {
		return err
	}

	// Down
	if err := g.SetKeybinding(bugTableView, 'j', gocui.ModNone,
		bt.cursorDown); err != nil {
		return err
	}
	if err := g.SetKeybinding(bugTableView, gocui.KeyArrowDown, gocui.ModNone,
		bt.cursorDown); err != nil {
		return err
	}
	// Up
	if err := g.SetKeybinding(bugTableView, 'k', gocui.ModNone,
		bt.cursorUp); err != nil {
		return err
	}
	if err := g.SetKeybinding(bugTableView, gocui.KeyArrowUp, gocui.ModNone,
		bt.cursorUp); err != nil {
		return err
	}

	// Previous page
	if err := g.SetKeybinding(bugTableView, 'h', gocui.ModNone,
		bt.previousPage); err != nil {
		return err
	}
	if err := g.SetKeybinding(bugTableView, gocui.KeyArrowLeft, gocui.ModNone,
		bt.previousPage); err != nil {
		return err
	}
	if err := g.SetKeybinding(bugTableView, gocui.KeyPgup, gocui.ModNone,
		bt.previousPage); err != nil {
		return err
	}
	// Next page
	if err := g.SetKeybinding(bugTableView, 'l', gocui.ModNone,
		bt.nextPage); err != nil {
		return err
	}
	if err := g.SetKeybinding(bugTableView, gocui.KeyArrowRight, gocui.ModNone,
		bt.nextPage); err != nil {
		return err
	}
	if err := g.SetKeybinding(bugTableView, gocui.KeyPgdn, gocui.ModNone,
		bt.nextPage); err != nil {
		return err
	}

	// New bug
	if err := g.SetKeybinding(bugTableView, 'n', gocui.ModNone,
		newBugWithEditor); err != nil {
		return err
	}

	return nil
}

func (bt *bugTable) paginate(max int) error {
	allIds, err := bt.cache.AllBugIds()
	if err != nil {
		return err
	}

	bt.allIds = allIds

	return bt.doPaginate(allIds, max)
}

func (bt *bugTable) doPaginate(allIds []string, max int) error {
	// clamp the cursor
	bt.cursor = maxInt(bt.cursor, 0)
	bt.cursor = minInt(bt.cursor, len(allIds)-1)

	nb := minInt(len(allIds)-bt.cursor, max)

	// slice the data
	ids := allIds[bt.cursor : bt.cursor+nb]

	bt.bugs = make([]*bug.Snapshot, len(ids))

	for i, id := range ids {
		b, err := bt.cache.ResolveBug(id)
		if err != nil {
			return err
		}

		bt.bugs[i] = b.Snapshot()
	}

	return nil
}

func (bt *bugTable) getTableLength() int {
	return len(bt.bugs)
}

func (bt *bugTable) getColumnWidths(maxX int) map[string]int {
	m := make(map[string]int)
	m["id"] = 10
	m["status"] = 8

	left := maxX - m["id"] - m["status"]

	m["summary"] = maxInt(30, left/3)
	left -= m["summary"]

	m["author"] = maxInt(left*2/5, 15)
	m["title"] = maxInt(left-m["author"], 10)

	return m
}

func (bt *bugTable) render(v *gocui.View, maxX int) {
	columnWidths := bt.getColumnWidths(maxX)

	for _, b := range bt.bugs {
		person := bug.Person{}
		if len(b.Comments) > 0 {
			create := b.Comments[0]
			person = create.Author
		}

		id := util.LeftPaddedString(b.HumanId(), columnWidths["id"], 2)
		status := util.LeftPaddedString(b.Status.String(), columnWidths["status"], 2)
		title := util.LeftPaddedString(b.Title, columnWidths["title"], 2)
		author := util.LeftPaddedString(person.Name, columnWidths["author"], 2)
		summary := util.LeftPaddedString(b.Summary(), columnWidths["summary"], 2)

		fmt.Fprintf(v, "%s %s %s %s %s\n", id, status, title, author, summary)
	}
}

func (bt *bugTable) renderHeader(v *gocui.View, maxX int) {
	columnWidths := bt.getColumnWidths(maxX)

	id := util.LeftPaddedString("ID", columnWidths["id"], 2)
	status := util.LeftPaddedString("STATUS", columnWidths["status"], 2)
	title := util.LeftPaddedString("TITLE", columnWidths["title"], 2)
	author := util.LeftPaddedString("AUTHOR", columnWidths["author"], 2)
	summary := util.LeftPaddedString("SUMMARY", columnWidths["summary"], 2)

	fmt.Fprintf(v, "\n")
	fmt.Fprintf(v, "%s %s %s %s %s\n", id, status, title, author, summary)

}

func (bt *bugTable) renderFooter(v *gocui.View, maxX int) {
	fmt.Fprintf(v, "Showing %d of %d bugs", len(bt.bugs), len(bt.allIds))
}

func (bt *bugTable) cursorDown(g *gocui.Gui, v *gocui.View) error {
	_, y := v.Cursor()
	y = minInt(y+1, bt.getTableLength()-1)

	err := v.SetCursor(0, y)
	if err != nil {
		return err
	}

	return nil
}

func (bt *bugTable) cursorUp(g *gocui.Gui, v *gocui.View) error {
	_, y := v.Cursor()
	y = maxInt(y-1, 0)

	err := v.SetCursor(0, y)
	if err != nil {
		return err
	}

	return nil
}

func (bt *bugTable) cursorClamp(v *gocui.View) error {
	_, y := v.Cursor()

	y = minInt(y, bt.getTableLength()-1)
	y = maxInt(y, 0)

	err := v.SetCursor(0, y)
	if err != nil {
		return err
	}

	return nil
}

func (bt *bugTable) nextPage(g *gocui.Gui, v *gocui.View) error {
	_, max := v.Size()

	allIds, err := bt.cache.AllBugIds()
	if err != nil {
		return err
	}

	bt.allIds = allIds

	if bt.cursor+max >= len(allIds) {
		return nil
	}

	bt.cursor += max

	return bt.doPaginate(allIds, max)
}

func (bt *bugTable) previousPage(g *gocui.Gui, v *gocui.View) error {
	_, max := v.Size()
	allIds, err := bt.cache.AllBugIds()
	if err != nil {
		return err
	}

	bt.allIds = allIds

	bt.cursor = maxInt(0, bt.cursor-max)

	return bt.doPaginate(allIds, max)
}
