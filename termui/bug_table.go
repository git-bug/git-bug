package termui

import (
	"bytes"
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

const remote = "origin"

type bugTable struct {
	repo         *cache.RepoCache
	query        *cache.Query
	allIds       []string
	bugs         []*cache.BugCache
	pageCursor   int
	selectCursor int
}

func newBugTable(c *cache.RepoCache) *bugTable {
	return &bugTable{
		repo: c,
		query: &cache.Query{
			OrderBy:        cache.OrderByCreation,
			OrderDirection: cache.OrderAscending,
		},
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

		fmt.Fprintf(v, "[Esc] Quit [←↓↑→,hjkl] Navigation [enter] Open bug [n] New bug [i] Pull [o] Push")
	}

	_, err = g.SetCurrentView(bugTableView)
	return err
}

func (bt *bugTable) keybindings(g *gocui.Gui) error {
	// Quit
	if err := g.SetKeybinding(bugTableView, gocui.KeyEsc, gocui.ModNone, quit); err != nil {
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

	// Pull
	if err := g.SetKeybinding(bugTableView, 'i', gocui.ModNone,
		bt.pull); err != nil {
		return err
	}

	// Push
	if err := g.SetKeybinding(bugTableView, 'o', gocui.ModNone,
		bt.push); err != nil {
		return err
	}

	return nil
}

func (bt *bugTable) disable(g *gocui.Gui) error {
	if err := g.DeleteView(bugTableView); err != nil && err != gocui.ErrUnknownView {
		return err
	}
	if err := g.DeleteView(bugTableHeaderView); err != nil && err != gocui.ErrUnknownView {
		return err
	}
	if err := g.DeleteView(bugTableFooterView); err != nil && err != gocui.ErrUnknownView {
		return err
	}
	if err := g.DeleteView(bugTableInstructionView); err != nil && err != gocui.ErrUnknownView {
		return err
	}
	return nil
}

func (bt *bugTable) paginate(max int) error {
	bt.allIds = bt.repo.QueryBugs(bt.query)

	return bt.doPaginate(max)
}

func (bt *bugTable) doPaginate(max int) error {
	// clamp the cursor
	bt.pageCursor = maxInt(bt.pageCursor, 0)
	bt.pageCursor = minInt(bt.pageCursor, len(bt.allIds))

	nb := minInt(len(bt.allIds)-bt.pageCursor, max)

	if nb < 0 {
		bt.bugs = []*cache.BugCache{}
		return nil
	}

	// slice the data
	ids := bt.allIds[bt.pageCursor : bt.pageCursor+nb]

	bt.bugs = make([]*cache.BugCache, len(ids))

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
		lastEdit := util.LeftPaddedString(humanize.Time(snap.LastEditTime()), columnWidths["lastEdit"], 2)

		fmt.Fprintf(v, "%s %s %s %s %s %s\n",
			util.Cyan(id),
			util.Yellow(status),
			title,
			util.Magenta(author),
			summary,
			lastEdit,
		)
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

	if bt.pageCursor+max >= len(bt.allIds) {
		return nil
	}

	bt.pageCursor += max

	return bt.doPaginate(max)
}

func (bt *bugTable) previousPage(g *gocui.Gui, v *gocui.View) error {
	_, max := v.Size()

	bt.pageCursor = maxInt(0, bt.pageCursor-max)

	return bt.doPaginate(max)
}

func (bt *bugTable) newBug(g *gocui.Gui, v *gocui.View) error {
	return newBugWithEditor(bt.repo)
}

func (bt *bugTable) openBug(g *gocui.Gui, v *gocui.View) error {
	_, y := v.Cursor()
	ui.showBug.SetBug(bt.bugs[y])
	return ui.activateWindow(ui.showBug)
}

func (bt *bugTable) pull(g *gocui.Gui, v *gocui.View) error {
	// Note: this is very hacky

	ui.msgPopup.Activate("Pull from remote "+remote, "...")

	go func() {
		stdout, err := bt.repo.Fetch(remote)

		if err != nil {
			g.Update(func(gui *gocui.Gui) error {
				ui.msgPopup.Activate(msgPopupErrorTitle, err.Error())
				return nil
			})
		} else {
			g.Update(func(gui *gocui.Gui) error {
				ui.msgPopup.UpdateMessage(stdout)
				return nil
			})
		}

		var buffer bytes.Buffer
		beginLine := ""

		for merge := range bt.repo.MergeAll(remote) {
			if merge.Status == bug.MsgMergeNothing {
				continue
			}

			if merge.Err != nil {
				g.Update(func(gui *gocui.Gui) error {
					ui.msgPopup.Activate(msgPopupErrorTitle, err.Error())
					return nil
				})
			} else {
				fmt.Fprintf(&buffer, "%s%s: %s",
					beginLine, util.Cyan(merge.Bug.HumanId()), merge.Status,
				)

				beginLine = "\n"

				g.Update(func(gui *gocui.Gui) error {
					ui.msgPopup.UpdateMessage(buffer.String())
					return nil
				})
			}
		}

		fmt.Fprintf(&buffer, "%sdone", beginLine)

		g.Update(func(gui *gocui.Gui) error {
			ui.msgPopup.UpdateMessage(buffer.String())
			return nil
		})

	}()

	return nil
}

func (bt *bugTable) push(g *gocui.Gui, v *gocui.View) error {
	ui.msgPopup.Activate("Push to remote "+remote, "...")

	go func() {
		// TODO: make the remote configurable
		stdout, err := bt.repo.Push(remote)

		if err != nil {
			g.Update(func(gui *gocui.Gui) error {
				ui.msgPopup.Activate(msgPopupErrorTitle, err.Error())
				return nil
			})
		} else {
			g.Update(func(gui *gocui.Gui) error {
				ui.msgPopup.UpdateMessage(stdout)
				return nil
			})
		}
	}()

	return nil
}
