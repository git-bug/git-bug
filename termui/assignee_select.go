package termui

import (
	"errors"
	"fmt"

	"github.com/awesome-gocui/gocui"

	"github.com/git-bug/git-bug/cache"
)

const assigneeSelectView = "assigneeSelectView"
const assigneeSelectInstructionsView = "assigneeSelectInstructionsView"

var assigneeSelectHelp = helpBar{
	{"q/Esc", "Cancel"},
	{"↓↑,jk", "Nav"},
	{"Enter", "Select"},
	{"u", "Unassign"},
}

type assigneeSelect struct {
	cache      *cache.RepoCache
	bug        *cache.BugCache
	identities []*cache.IdentityExcerpt
	selected   int
	scroll     int
	childViews []string
}

func newAssigneeSelect() *assigneeSelect {
	return &assigneeSelect{}
}

func (as *assigneeSelect) SetBug(c *cache.RepoCache, bug *cache.BugCache) {
	as.cache = c
	as.bug = bug

	ids := c.Identities().AllIds()
	as.identities = make([]*cache.IdentityExcerpt, 0, len(ids))
	for _, id := range ids {
		excerpt, err := c.Identities().ResolveExcerpt(id)
		if err == nil {
			as.identities = append(as.identities, excerpt)
		}
	}

	as.selected = 0
	as.scroll = 0
}

func (as *assigneeSelect) keybindings(g *gocui.Gui) error {
	// Abort
	if err := g.SetKeybinding(assigneeSelectView, gocui.KeyEsc, gocui.ModNone, as.abort); err != nil {
		return err
	}
	if err := g.SetKeybinding(assigneeSelectView, 'q', gocui.ModNone, as.abort); err != nil {
		return err
	}
	// Up
	if err := g.SetKeybinding(assigneeSelectView, gocui.KeyArrowUp, gocui.ModNone, as.selectPrevious); err != nil {
		return err
	}
	if err := g.SetKeybinding(assigneeSelectView, 'k', gocui.ModNone, as.selectPrevious); err != nil {
		return err
	}
	// Down
	if err := g.SetKeybinding(assigneeSelectView, gocui.KeyArrowDown, gocui.ModNone, as.selectNext); err != nil {
		return err
	}
	if err := g.SetKeybinding(assigneeSelectView, 'j', gocui.ModNone, as.selectNext); err != nil {
		return err
	}
	// Select
	if err := g.SetKeybinding(assigneeSelectView, gocui.KeyEnter, gocui.ModNone, as.selectAssignee); err != nil {
		return err
	}
	// Unassign
	if err := g.SetKeybinding(assigneeSelectView, 'u', gocui.ModNone, as.unassign); err != nil {
		return err
	}
	return nil
}

func (as *assigneeSelect) layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()
	as.childViews = nil

	width := 40
	height := minInt(len(as.identities)+2, maxY-4)
	x0 := maxX/2 - width/2
	y0 := maxY/2 - height/2

	v, err := g.SetView(assigneeSelectView, x0, y0, x0+width, y0+height, 0)
	if err != nil {
		if !errors.Is(err, gocui.ErrUnknownView) {
			return err
		}
		as.childViews = append(as.childViews, assigneeSelectView)
		v.Frame = true
		v.Title = " Select Assignee "
	}

	_, viewHeight := v.Size()
	v.Clear()
	as.renderIdentities(v, viewHeight)

	v, err = g.SetView(assigneeSelectInstructionsView, -1, maxY-2, maxX, maxY, 0)
	if err != nil {
		if !errors.Is(err, gocui.ErrUnknownView) {
			return err
		}
		as.childViews = append(as.childViews, assigneeSelectInstructionsView)
		v.Frame = false
		v.FgColor = gocui.ColorWhite
	}

	v.Clear()
	_, _ = fmt.Fprint(v, assigneeSelectHelp.Render(maxX))

	_, err = g.SetCurrentView(assigneeSelectView)
	return err
}

func (as *assigneeSelect) renderIdentities(v *gocui.View, viewHeight int) {
	for i, identity := range as.identities {
		if i < as.scroll || i >= as.scroll+viewHeight {
			continue
		}
		marker := "  "
		if i == as.selected {
			marker = "> "
		}
		currentMarker := ""
		if as.bug.Snapshot().Assignee != nil && as.bug.Snapshot().Assignee.Id() == identity.Id() {
			currentMarker = " (current)"
		}
		_, _ = fmt.Fprintf(v, "%s%s%s\n", marker, identity.DisplayName(), currentMarker)
	}
}

func (as *assigneeSelect) disable(g *gocui.Gui) error {
	for _, view := range as.childViews {
		if err := g.DeleteView(view); err != nil && !errors.Is(err, gocui.ErrUnknownView) {
			return err
		}
	}
	return nil
}

func (as *assigneeSelect) selectPrevious(g *gocui.Gui, v *gocui.View) error {
	if as.selected > 0 {
		as.selected--
	}
	_, viewHeight := v.Size()
	if as.selected < as.scroll {
		as.scroll = as.selected
	}
	if as.selected >= as.scroll+viewHeight {
		as.scroll = as.selected - viewHeight + 1
	}
	return nil
}

func (as *assigneeSelect) selectNext(g *gocui.Gui, v *gocui.View) error {
	if as.selected < len(as.identities)-1 {
		as.selected++
	}
	_, viewHeight := v.Size()
	if as.selected >= as.scroll+viewHeight {
		as.scroll = as.selected - viewHeight + 1
	}
	return nil
}

func (as *assigneeSelect) selectAssignee(g *gocui.Gui, v *gocui.View) error {
	if as.selected >= 0 && as.selected < len(as.identities) {
		excerpt := as.identities[as.selected]
		identity, err := as.cache.Identities().Resolve(excerpt.Id())
		if err != nil {
			return err
		}
		_, err = as.bug.SetAssignee(identity.Id(), identity)
		if err != nil {
			return err
		}
	}
	return ui.activateWindow(ui.showBug)
}

func (as *assigneeSelect) unassign(g *gocui.Gui, v *gocui.View) error {
	_, err := as.bug.Unassign()
	if err != nil {
		return err
	}
	return ui.activateWindow(ui.showBug)
}

func (as *assigneeSelect) abort(g *gocui.Gui, v *gocui.View) error {
	return ui.activateWindow(ui.showBug)
}
