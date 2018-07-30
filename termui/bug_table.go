package termui

import (
	"fmt"
	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/jroimartin/gocui"
)

type bugTable struct {
	cache  cache.RepoCacher
	bugs   []*bug.Snapshot
	cursor int
}

func newBugTable(cache cache.RepoCacher) *bugTable {
	return &bugTable{
		cache:  cache,
		cursor: 0,
	}
}

func (bt *bugTable) paginate(max int) error {
	allIds, err := bt.cache.AllBugIds()
	if err != nil {
		return err
	}

	return bt.doPaginate(allIds, max)
}

func (bt *bugTable) nextPage(max int) error {
	allIds, err := bt.cache.AllBugIds()
	if err != nil {
		return err
	}

	if bt.cursor+max > len(allIds) {
		return nil
	}

	bt.cursor += max

	return bt.doPaginate(allIds, max)
}

func (bt *bugTable) previousPage(max int) error {
	allIds, err := bt.cache.AllBugIds()
	if err != nil {
		return err
	}

	bt.cursor = maxInt(0, bt.cursor-max)

	return bt.doPaginate(allIds, max)
}

func (bt *bugTable) doPaginate(allIds []string, max int) error {
	// clamp the cursor
	bt.cursor = maxInt(bt.cursor, 0)
	bt.cursor = minInt(bt.cursor, len(allIds)-1)

	// slice the data
	nb := minInt(len(allIds)-bt.cursor, max)

	ids := allIds[bt.cursor:nb]

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
	ui.bugTable.renderHeader(v, maxX)

	v, err = g.SetView("table", -1, 1, maxX, maxY-2)

	if err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}

		v.Frame = false
		v.Highlight = true
		v.SelBgColor = gocui.ColorWhite
		v.SelFgColor = gocui.ColorBlack

		_, err = g.SetCurrentView("table")

		if err != nil {
			return err
		}
	}

	_, tableHeight := v.Size()
	err = bt.paginate(tableHeight)
	if err != nil {
		return err
	}

	v.Clear()
	ui.bugTable.render(v, maxX)

	v, err = g.SetView("footer", -1, maxY-3, maxX, maxY)

	if err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}

		v.Frame = false
	}

	v.Clear()
	ui.bugTable.renderFooter(v, maxX)

	v, err = g.SetView("instructions", -1, maxY-2, maxX, maxY)

	if err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}

		v.Frame = false
		v.BgColor = gocui.ColorBlue

		fmt.Fprintf(v, "[q] Quit [h] Go back [j] Down [k] Up [l] Go forward [m] Load Additional [p] Play [enter] Play and Exit")
	}

	return nil
}

func (bt *bugTable) getTableLength() int {
	return len(bt.bugs)
}

func (bt *bugTable) render(v *gocui.View, maxX int) {
	for _, b := range bt.bugs {
		fmt.Fprintln(v, b.Title)
	}
}

func (bt *bugTable) renderHeader(v *gocui.View, maxX int) {
	fmt.Fprintf(v, "header")

}

func (bt *bugTable) renderFooter(v *gocui.View, maxX int) {
	fmt.Fprintf(v, "footer")
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
