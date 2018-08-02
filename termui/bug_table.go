package termui

import (
	"fmt"
	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/util"
	"github.com/dustin/go-humanize"
	"github.com/jroimartin/gocui"
)

const bugTableView = "bugTableView"
const bugTableHeaderView = "bugTableHeaderView"
const bugTableFooterView = "bugTableFooterView"
const bugTableInstructionView = "bugTableInstructionView"

type bugTable struct {
	repo         cache.RepoCacher
	allIds       []string
	bugs         []cache.BugCacher
	pageCursor   int
	selectCursor int
}

func newBugTable(cache cache.RepoCacher) *bugTable {
	return &bugTable{
		repo:         cache,
		pageCursor:   0,
		selectCursor: 0,
	}
}

func (bt *bugTable) layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()

	if maxY < 4 {
		// window too small !
		return nil
	}

	v, err := g.SetView(bugTableHeaderView, -1, -1, maxX, 3)

	if err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}

		v.Frame = false
	}

	v.Clear()
	bt.renderHeader(v, maxX)

	v, err = g.SetView(bugTableView, -1, 1, maxX, maxY-3)

	if err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}

		v.Frame = false
		v.Highlight = true
		v.SelBgColor = gocui.ColorWhite
		v.SelFgColor = gocui.ColorBlack

		// restore the cursor
		// window is too small to set the cursor properly, ignoring the error
		_ = v.SetCursor(0, bt.selectCursor)
	}

	_, viewHeight := v.Size()
	err = bt.paginate(viewHeight)
	if err != nil {
		return err
	}

	err = bt.cursorClamp(v)
	if err != nil {
		return err
	}

	v.Clear()
	bt.render(v, maxX)

	v, err = g.SetView(bugTableFooterView, -1, maxY-4, maxX, maxY)

	if err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}

		v.Frame = false
	}

	v.Clear()
	bt.renderFooter(v, maxX)

	v, err = g.SetView(bugTableInstructionView, -1, maxY-2, maxX, maxY)

	if err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}

		v.Frame = false
		v.BgColor = gocui.ColorBlue

		fmt.Fprintf(v, "[q] Quit [←,h] Previous page [↓,j] Down [↑,k] Up [→,l] Next page [enter] Open bug [n] New bug")
	}

	_, err = g.SetCurrentView(bugTableView)
	return err
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
		bt.newBug); err != nil {
		return err
	}

	// Open bug
	if err := g.SetKeybinding(bugTableView, gocui.KeyEnter, gocui.ModNone,
		bt.openBug); err != nil {
		return err
	}

	return nil
}

func (bt *bugTable) disable(g *gocui.Gui) error {
	if err := g.DeleteView(bugTableView); err != nil {
		return err
	}
	if err := g.DeleteView(bugTableHeaderView); err != nil {
		return err
	}
	if err := g.DeleteView(bugTableFooterView); err != nil {
		return err
	}
	if err := g.DeleteView(bugTableInstructionView); err != nil {
		return err
	}
	return nil
}

func (bt *bugTable) paginate(max int) error {
	allIds, err := bt.repo.AllBugIds()
	if err != nil {
		return err
	}

	bt.allIds = allIds

	return bt.doPaginate(allIds, max)
}

func (bt *bugTable) doPaginate(allIds []string, max int) error {
	// clamp the cursor
	bt.pageCursor = maxInt(bt.pageCursor, 0)
	bt.pageCursor = minInt(bt.pageCursor, len(allIds)-1)

	nb := minInt(len(allIds)-bt.pageCursor, max)

	if nb < 0 {
		bt.bugs = []cache.BugCacher{}
		return nil
	}

	// slice the data
	ids := allIds[bt.pageCursor : bt.pageCursor+nb]

	bt.bugs = make([]cache.BugCacher, len(ids))

	for i, id := range ids {
		b, err := bt.repo.ResolveBug(id)
		if err != nil {
			return err
		}

		bt.bugs[i] = b
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

	left := maxX - 5 - m["id"] - m["status"]

	m["summary"] = maxInt(11, left/6)
	left -= m["summary"]

	m["lastEdit"] = maxInt(19, left/6)
	left -= m["lastEdit"]

	m["author"] = maxInt(left*2/5, 15)
	m["title"] = maxInt(left-m["author"], 10)

	return m
}

func (bt *bugTable) render(v *gocui.View, maxX int) {
	columnWidths := bt.getColumnWidths(maxX)

	for _, b := range bt.bugs {
		person := bug.Person{}
		snap := b.Snapshot()
		if len(snap.Comments) > 0 {
			create := snap.Comments[0]
			person = create.Author
		}

		id := util.LeftPaddedString(snap.HumanId(), columnWidths["id"], 2)
		status := util.LeftPaddedString(snap.Status.String(), columnWidths["status"], 2)
		title := util.LeftPaddedString(snap.Title, columnWidths["title"], 2)
		author := util.LeftPaddedString(person.Name, columnWidths["author"], 2)
		summary := util.LeftPaddedString(snap.Summary(), columnWidths["summary"], 2)
		lastEdit := util.LeftPaddedString(humanize.Time(snap.LastEdit()), columnWidths["lastEdit"], 2)

		fmt.Fprintf(v, "%s %s %s %s %s %s\n", id, status, title, author, summary, lastEdit)
	}
}

func (bt *bugTable) renderHeader(v *gocui.View, maxX int) {
	columnWidths := bt.getColumnWidths(maxX)

	id := util.LeftPaddedString("ID", columnWidths["id"], 2)
	status := util.LeftPaddedString("STATUS", columnWidths["status"], 2)
	title := util.LeftPaddedString("TITLE", columnWidths["title"], 2)
	author := util.LeftPaddedString("AUTHOR", columnWidths["author"], 2)
	summary := util.LeftPaddedString("SUMMARY", columnWidths["summary"], 2)
	lastEdit := util.LeftPaddedString("LAST EDIT", columnWidths["lastEdit"], 2)

	fmt.Fprintf(v, "\n")
	fmt.Fprintf(v, "%s %s %s %s %s %s\n", id, status, title, author, summary, lastEdit)

}

func (bt *bugTable) renderFooter(v *gocui.View, maxX int) {
	fmt.Fprintf(v, " \nShowing %d of %d bugs", len(bt.bugs), len(bt.allIds))
}

func (bt *bugTable) cursorDown(g *gocui.Gui, v *gocui.View) error {
	_, y := v.Cursor()
	y = minInt(y+1, bt.getTableLength()-1)

	// window is too small to set the cursor properly, ignoring the error
	_ = v.SetCursor(0, y)
	bt.selectCursor = y

	return nil
}

func (bt *bugTable) cursorUp(g *gocui.Gui, v *gocui.View) error {
	_, y := v.Cursor()
	y = maxInt(y-1, 0)

	// window is too small to set the cursor properly, ignoring the error
	_ = v.SetCursor(0, y)
	bt.selectCursor = y

	return nil
}

func (bt *bugTable) cursorClamp(v *gocui.View) error {
	_, y := v.Cursor()

	y = minInt(y, bt.getTableLength()-1)
	y = maxInt(y, 0)

	// window is too small to set the cursor properly, ignoring the error
	_ = v.SetCursor(0, y)
	bt.selectCursor = y

	return nil
}

func (bt *bugTable) nextPage(g *gocui.Gui, v *gocui.View) error {
	_, max := v.Size()

	allIds, err := bt.repo.AllBugIds()
	if err != nil {
		return err
	}

	bt.allIds = allIds

	if bt.pageCursor+max >= len(allIds) {
		return nil
	}

	bt.pageCursor += max

	return bt.doPaginate(allIds, max)
}

func (bt *bugTable) previousPage(g *gocui.Gui, v *gocui.View) error {
	_, max := v.Size()
	allIds, err := bt.repo.AllBugIds()
	if err != nil {
		return err
	}

	bt.allIds = allIds

	bt.pageCursor = maxInt(0, bt.pageCursor-max)

	return bt.doPaginate(allIds, max)
}

func (bt *bugTable) newBug(g *gocui.Gui, v *gocui.View) error {
	return newBugWithEditor(bt.repo)
}

func (bt *bugTable) openBug(g *gocui.Gui, v *gocui.View) error {
	_, y := v.Cursor()
	ui.showBug.bug = bt.bugs[bt.pageCursor+y]
	return ui.activateWindow(ui.showBug)
}
