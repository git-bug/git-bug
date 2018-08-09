package termui

import (
	"fmt"

	"github.com/MichaelMure/git-bug/bug/operations"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/util"
	"github.com/jroimartin/gocui"
)

const showBugView = "showBugView"
const showBugSidebarView = "showBugSidebarView"
const showBugInstructionView = "showBugInstructionView"
const showBugHeaderView = "showBugHeaderView"

const timeLayout = "Jan 2 2006"

type showBug struct {
	cache          cache.RepoCacher
	bug            cache.BugCacher
	childViews     []string
	selectableView []string
	selected       string
	scroll         int
}

func newShowBug(cache cache.RepoCacher) *showBug {
	return &showBug{
		cache: cache,
	}
}

func (sb *showBug) layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()

	v, err := g.SetView(showBugView, 0, 0, maxX*2/3, maxY-2)

	if err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}

		sb.childViews = append(sb.childViews, showBugView)
		v.Frame = false
	}

	v.Clear()
	err = sb.renderMain(g, v)
	if err != nil {
		return err
	}

	v, err = g.SetView(showBugSidebarView, maxX*2/3+1, 0, maxX-1, maxY-2)

	if err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}

		sb.childViews = append(sb.childViews, showBugSidebarView)
		v.Frame = true
	}

	v.Clear()
	sb.renderSidebar(v)

	v, err = g.SetView(showBugInstructionView, -1, maxY-2, maxX, maxY)

	if err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}

		sb.childViews = append(sb.childViews, showBugInstructionView)
		v.Frame = false
		v.BgColor = gocui.ColorBlue

		fmt.Fprintf(v, "[q] Save and return [c] Comment [t] Change title [↓,j] Down [↑,k] Up")
	}

	_, err = g.SetCurrentView(showBugView)
	return err
}

func (sb *showBug) keybindings(g *gocui.Gui) error {
	// Return
	if err := g.SetKeybinding(showBugView, 'q', gocui.ModNone, sb.saveAndBack); err != nil {
		return err
	}

	// Scrolling
	if err := g.SetKeybinding(showBugView, gocui.KeyPgup, gocui.ModNone,
		sb.scrollUp); err != nil {
		return err
	}
	if err := g.SetKeybinding(showBugView, gocui.KeyPgdn, gocui.ModNone,
		sb.scrollDown); err != nil {
		return err
	}

	// Down
	if err := g.SetKeybinding(showBugView, 'j', gocui.ModNone,
		sb.selectNext); err != nil {
		return err
	}
	if err := g.SetKeybinding(showBugView, gocui.KeyArrowDown, gocui.ModNone,
		sb.selectNext); err != nil {
		return err
	}
	// Up
	if err := g.SetKeybinding(showBugView, 'k', gocui.ModNone,
		sb.selectPrevious); err != nil {
		return err
	}
	if err := g.SetKeybinding(showBugView, gocui.KeyArrowUp, gocui.ModNone,
		sb.selectPrevious); err != nil {
		return err
	}

	// Comment
	if err := g.SetKeybinding(showBugView, 'c', gocui.ModNone,
		sb.comment); err != nil {
		return err
	}

	// Title
	if err := g.SetKeybinding(showBugView, 't', gocui.ModNone,
		sb.setTitle); err != nil {
		return err
	}

	// Labels

	return nil
}

func (sb *showBug) disable(g *gocui.Gui) error {
	for _, view := range sb.childViews {
		if err := g.DeleteView(view); err != nil {
			return err
		}
	}
	return nil
}

func (sb *showBug) renderMain(g *gocui.Gui, mainView *gocui.View) error {
	maxX, _ := mainView.Size()
	x0, y0, _, _, _ := g.ViewPosition(mainView.Name())

	y0 -= sb.scroll

	snap := sb.bug.Snapshot()

	sb.childViews = nil
	sb.selectableView = nil

	header := fmt.Sprintf("[ %s ] %s\n\n[ %s ] %s opened this bug on %s",
		util.Cyan(snap.HumanId()),
		util.Bold(snap.Title),
		util.Yellow(snap.Status),
		util.Magenta(snap.Author.Name),
		snap.CreatedAt.Format(timeLayout),
	)
	content, lines := util.TextWrap(header, maxX)

	v, err := sb.createOpView(g, showBugHeaderView, x0, y0, maxX+1, lines, false)
	if err != nil {
		return err
	}

	fmt.Fprint(v, content)
	y0 += lines + 1

	for i, op := range snap.Operations {
		viewName := fmt.Sprintf("op%d", i)

		// TODO: me might skip the rendering of blocks that are outside of the view
		// but to do that we need to rework how sb.selectableView is maintained

		switch op.(type) {

		case operations.CreateOperation:
			create := op.(operations.CreateOperation)
			content, lines := util.TextWrap(create.Message, maxX)

			v, err := sb.createOpView(g, viewName, x0, y0, maxX+1, lines, true)
			if err != nil {
				return err
			}
			fmt.Fprint(v, content)
			y0 += lines + 2

		case operations.AddCommentOperation:
			comment := op.(operations.AddCommentOperation)

			content := fmt.Sprintf("%s commented on %s\n\n%s",
				util.Magenta(comment.Author.Name),
				comment.Time().Format(timeLayout),
				comment.Message,
			)
			content, lines := util.TextWrapPadded(content, maxX, 6)

			v, err := sb.createOpView(g, viewName, x0, y0, maxX+1, lines, true)
			if err != nil {
				return err
			}
			fmt.Fprint(v, content)
			y0 += lines + 2

		case operations.SetTitleOperation:
			setTitle := op.(operations.SetTitleOperation)

			content := fmt.Sprintf("%s changed the title to %s on %s",
				util.Magenta(setTitle.Author.Name),
				util.Bold(setTitle.Title),
				setTitle.Time().Format(timeLayout),
			)
			content, lines := util.TextWrap(content, maxX)

			v, err := sb.createOpView(g, viewName, x0, y0, maxX+1, lines, true)
			if err != nil {
				return err
			}
			fmt.Fprint(v, content)
			y0 += lines + 2
		}
	}

	return nil
}

func (sb *showBug) createOpView(g *gocui.Gui, name string, x0 int, y0 int, maxX int, height int, selectable bool) (*gocui.View, error) {
	v, err := g.SetView(name, x0, y0, maxX, y0+height+1)

	if err != nil && err != gocui.ErrUnknownView {
		return nil, err
	}

	sb.childViews = append(sb.childViews, name)

	if selectable {
		sb.selectableView = append(sb.selectableView, name)
	}

	v.Frame = sb.selected == name

	v.Clear()

	return v, nil
}

func (sb *showBug) renderSidebar(v *gocui.View) {
	maxX, _ := v.Size()
	snap := sb.bug.Snapshot()

	title := util.LeftPaddedString("LABEL", maxX, 2)
	fmt.Fprintf(v, title+"\n\n")

	for _, label := range snap.Labels {
		fmt.Fprintf(v, util.LeftPaddedString(label.String(), maxX, 2))
		fmt.Fprintln(v)
	}
}

func (sb *showBug) saveAndBack(g *gocui.Gui, v *gocui.View) error {
	err := sb.bug.CommitAsNeeded()
	if err != nil {
		return err
	}
	ui.activateWindow(ui.bugTable)
	return nil
}

func (sb *showBug) scrollUp(g *gocui.Gui, v *gocui.View) error {
	mainView, err := g.View(showBugView)
	if err != nil {
		return err
	}

	_, maxY := mainView.Size()

	sb.scroll -= maxY / 2

	sb.scroll = maxInt(sb.scroll, 0)

	return nil
}

func (sb *showBug) scrollDown(g *gocui.Gui, v *gocui.View) error {
	_, maxY := v.Size()

	lastViewName := sb.childViews[len(sb.childViews)-1]

	lastView, err := g.View(lastViewName)
	if err != nil {
		return err
	}

	_, vMaxY := lastView.Size()

	_, vy0, _, _, err := g.ViewPosition(lastViewName)
	if err != nil {
		return err
	}

	maxScroll := vy0 + sb.scroll + vMaxY - maxY

	sb.scroll += maxY / 2

	sb.scroll = minInt(sb.scroll, maxScroll)

	return nil
}

func (sb *showBug) selectPrevious(g *gocui.Gui, v *gocui.View) error {
	if len(sb.selectableView) == 0 {
		return nil
	}

	defer sb.focusView(g)

	for i, name := range sb.selectableView {
		if name == sb.selected {
			// special case to scroll up to the top
			if i == 0 {
				sb.scroll = 0
			}

			sb.selected = sb.selectableView[maxInt(i-1, 0)]
			return nil
		}
	}

	if sb.selected == "" {
		sb.selected = sb.selectableView[0]
	}

	return nil
}

func (sb *showBug) selectNext(g *gocui.Gui, v *gocui.View) error {
	if len(sb.selectableView) == 0 {
		return nil
	}

	defer sb.focusView(g)

	for i, name := range sb.selectableView {
		if name == sb.selected {
			sb.selected = sb.selectableView[minInt(i+1, len(sb.selectableView)-1)]
			return nil
		}
	}

	if sb.selected == "" {
		sb.selected = sb.selectableView[0]
	}

	return nil
}

func (sb *showBug) focusView(g *gocui.Gui) error {
	mainView, err := g.View(showBugView)
	if err != nil {
		return err
	}

	_, maxY := mainView.Size()

	_, vy0, _, _, err := g.ViewPosition(sb.selected)
	if err != nil {
		return err
	}

	v, err := g.View(sb.selected)
	if err != nil {
		return err
	}

	_, vMaxY := v.Size()

	vy1 := vy0 + vMaxY

	if vy0 < 0 {
		sb.scroll += vy0
		return nil
	}

	if vy1 > maxY {
		sb.scroll -= maxY - vy1
	}

	return nil
}

func (sb *showBug) comment(g *gocui.Gui, v *gocui.View) error {
	return addCommentWithEditor(sb.bug)
}

func (sb *showBug) setTitle(g *gocui.Gui, v *gocui.View) error {
	return setTitleWithEditor(sb.bug)
}
