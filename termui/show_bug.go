package termui

import (
	"bytes"
	"fmt"
	"strings"

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
	cache              *cache.RepoCache
	bug                *cache.BugCache
	childViews         []string
	mainSelectableView []string
	sideSelectableView []string
	selected           string
	isOnSide           bool
	scroll             int
}

func newShowBug(cache *cache.RepoCache) *showBug {
	return &showBug{
		cache: cache,
	}
}

func (sb *showBug) SetBug(bug *cache.BugCache) {
	sb.bug = bug
	sb.scroll = 0
	sb.selected = ""
	sb.isOnSide = false
}

func (sb *showBug) layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()
	sb.childViews = nil

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
		v.Frame = false
	}

	v.Clear()
	err = sb.renderSidebar(g, v)
	if err != nil {
		return err
	}

	v, err = g.SetView(showBugInstructionView, -1, maxY-2, maxX, maxY)

	if err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}

		sb.childViews = append(sb.childViews, showBugInstructionView)
		v.Frame = false
		v.BgColor = gocui.ColorBlue
	}

	v.Clear()
	fmt.Fprintf(v, "[q] Save and return [←↓↑→,hjkl] Navigation ")

	if sb.isOnSide {
		fmt.Fprint(v, "[a] Add label [r] Remove label")
	} else {
		fmt.Fprint(v, "[c] Comment [t] Change title")
	}

	_, err = g.SetViewOnTop(showBugInstructionView)
	if err != nil {
		return err
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

	// Left
	if err := g.SetKeybinding(showBugView, 'h', gocui.ModNone,
		sb.left); err != nil {
		return err
	}
	if err := g.SetKeybinding(showBugView, gocui.KeyArrowLeft, gocui.ModNone,
		sb.left); err != nil {
		return err
	}
	// Right
	if err := g.SetKeybinding(showBugView, 'l', gocui.ModNone,
		sb.right); err != nil {
		return err
	}
	if err := g.SetKeybinding(showBugView, gocui.KeyArrowRight, gocui.ModNone,
		sb.right); err != nil {
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
	if err := g.SetKeybinding(showBugView, 'a', gocui.ModNone,
		sb.addLabel); err != nil {
		return err
	}
	if err := g.SetKeybinding(showBugView, 'r', gocui.ModNone,
		sb.removeLabel); err != nil {
		return err
	}

	return nil
}

func (sb *showBug) disable(g *gocui.Gui) error {
	for _, view := range sb.childViews {
		if err := g.DeleteView(view); err != nil && err != gocui.ErrUnknownView {
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

	sb.mainSelectableView = nil

	bugHeader := fmt.Sprintf("[%s] %s\n\n[%s] %s opened this bug on %s",
		util.Cyan(snap.HumanId()),
		util.Bold(snap.Title),
		util.Yellow(snap.Status),
		util.Magenta(snap.Author.Name),
		snap.CreatedAt.Format(timeLayout),
	)
	bugHeader, lines := util.TextWrap(bugHeader, maxX)

	v, err := sb.createOpView(g, showBugHeaderView, x0, y0, maxX+1, lines, false)
	if err != nil {
		return err
	}

	fmt.Fprint(v, bugHeader)
	y0 += lines + 1

	for i, op := range snap.Operations {
		viewName := fmt.Sprintf("op%d", i)

		// TODO: me might skip the rendering of blocks that are outside of the view
		// but to do that we need to rework how sb.mainSelectableView is maintained

		switch op.(type) {

		case operations.CreateOperation:
			create := op.(operations.CreateOperation)
			content, lines := util.TextWrapPadded(create.Message, maxX, 4)

			v, err := sb.createOpView(g, viewName, x0, y0, maxX+1, lines, true)
			if err != nil {
				return err
			}
			fmt.Fprint(v, content)
			y0 += lines + 2

		case operations.AddCommentOperation:
			comment := op.(operations.AddCommentOperation)

			message, _ := util.TextWrapPadded(comment.Message, maxX, 4)
			content := fmt.Sprintf("%s commented on %s\n\n%s",
				util.Magenta(comment.Author.Name),
				comment.Time().Format(timeLayout),
				message,
			)
			content, lines = util.TextWrap(content, maxX)

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

		case operations.SetStatusOperation:
			setStatus := op.(operations.SetStatusOperation)

			content := fmt.Sprintf("%s %s the bug on %s",
				util.Magenta(setStatus.Author.Name),
				util.Bold(setStatus.Status.Action()),
				setStatus.Time().Format(timeLayout),
			)
			content, lines := util.TextWrap(content, maxX)

			v, err := sb.createOpView(g, viewName, x0, y0, maxX+1, lines, true)
			if err != nil {
				return err
			}
			fmt.Fprint(v, content)
			y0 += lines + 2

		case operations.LabelChangeOperation:
			labelChange := op.(operations.LabelChangeOperation)

			var added []string
			for _, label := range labelChange.Added {
				added = append(added, util.Bold("\""+label+"\""))
			}

			var removed []string
			for _, label := range labelChange.Removed {
				removed = append(removed, util.Bold("\""+label+"\""))
			}

			var action bytes.Buffer

			if len(added) > 0 {
				action.WriteString("added ")
				action.WriteString(strings.Join(added, ", "))

				if len(removed) > 0 {
					action.WriteString(" and ")
				}
			}

			if len(removed) > 0 {
				action.WriteString("removed ")
				action.WriteString(strings.Join(removed, ", "))
			}

			if len(added)+len(removed) > 1 {
				action.WriteString(" labels")
			} else {
				action.WriteString(" label")
			}

			content := fmt.Sprintf("%s %s on %s",
				util.Magenta(labelChange.Author.Name),
				action.String(),
				labelChange.Time().Format(timeLayout),
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
		sb.mainSelectableView = append(sb.mainSelectableView, name)
	}

	v.Frame = sb.selected == name

	v.Clear()

	return v, nil
}

func (sb *showBug) createSideView(g *gocui.Gui, name string, x0 int, y0 int, maxX int, height int) (*gocui.View, error) {
	v, err := g.SetView(name, x0, y0, maxX, y0+height+1)

	if err != nil && err != gocui.ErrUnknownView {
		return nil, err
	}

	sb.childViews = append(sb.childViews, name)
	sb.sideSelectableView = append(sb.sideSelectableView, name)

	v.Frame = sb.selected == name

	v.Clear()

	return v, nil
}

func (sb *showBug) renderSidebar(g *gocui.Gui, sideView *gocui.View) error {
	maxX, _ := sideView.Size()
	x0, y0, _, _, _ := g.ViewPosition(sideView.Name())
	maxX += x0

	snap := sb.bug.Snapshot()

	sb.sideSelectableView = nil

	labelStr := make([]string, len(snap.Labels))
	for i, l := range snap.Labels {
		labelStr[i] = string(l)
	}

	labels := strings.Join(labelStr, "\n")
	labels, lines := util.TextWrapPadded(labels, maxX, 2)

	content := fmt.Sprintf("%s\n\n%s", util.Bold("Labels"), labels)

	v, err := sb.createSideView(g, "sideLabels", x0, y0, maxX, lines+2)
	if err != nil {
		return err
	}

	fmt.Fprint(v, content)

	return nil
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

	lastViewName := sb.mainSelectableView[len(sb.mainSelectableView)-1]

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
	defer sb.focusView(g)

	var selectable []string
	if sb.isOnSide {
		selectable = sb.sideSelectableView
	} else {
		selectable = sb.mainSelectableView
	}

	for i, name := range selectable {
		if name == sb.selected {
			// special case to scroll up to the top
			if i == 0 {
				sb.scroll = 0
			}

			sb.selected = selectable[maxInt(i-1, 0)]
			return nil
		}
	}

	if sb.selected == "" && len(selectable) > 0 {
		sb.selected = selectable[0]
	}

	return nil
}

func (sb *showBug) selectNext(g *gocui.Gui, v *gocui.View) error {
	defer sb.focusView(g)

	var selectable []string
	if sb.isOnSide {
		selectable = sb.sideSelectableView
	} else {
		selectable = sb.mainSelectableView
	}

	for i, name := range selectable {
		if name == sb.selected {
			sb.selected = selectable[minInt(i+1, len(selectable)-1)]
			return nil
		}
	}

	if sb.selected == "" && len(selectable) > 0 {
		sb.selected = selectable[0]
	}

	return nil
}

func (sb *showBug) left(g *gocui.Gui, v *gocui.View) error {
	if sb.isOnSide {
		sb.isOnSide = false
		sb.selected = ""
		return sb.selectNext(g, v)
	}

	if sb.selected == "" {
		return sb.selectNext(g, v)
	}

	return nil
}

func (sb *showBug) right(g *gocui.Gui, v *gocui.View) error {
	if !sb.isOnSide {
		sb.isOnSide = true
		sb.selected = ""
		return sb.selectNext(g, v)
	}

	if sb.selected == "" {
		return sb.selectNext(g, v)
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

func (sb *showBug) addLabel(g *gocui.Gui, v *gocui.View) error {
	c := ui.inputPopup.Activate("Add labels")

	go func() {
		input := <-c

		labels := strings.FieldsFunc(input, func(r rune) bool {
			return r == ' ' || r == ','
		})

		err := sb.bug.ChangeLabels(trimLabels(labels), nil)
		if err != nil {
			ui.msgPopup.Activate(msgPopupErrorTitle, err.Error())
		}

		g.Update(func(gui *gocui.Gui) error {
			return nil
		})
	}()

	return nil
}

func (sb *showBug) removeLabel(g *gocui.Gui, v *gocui.View) error {
	c := ui.inputPopup.Activate("Remove labels")

	go func() {
		input := <-c

		labels := strings.FieldsFunc(input, func(r rune) bool {
			return r == ' ' || r == ','
		})

		err := sb.bug.ChangeLabels(nil, trimLabels(labels))
		if err != nil {
			ui.msgPopup.Activate(msgPopupErrorTitle, err.Error())
		}

		g.Update(func(gui *gocui.Gui) error {
			return nil
		})
	}()

	return nil
}

func trimLabels(labels []string) []string {
	var result []string

	for _, label := range labels {
		trimmed := strings.TrimSpace(label)
		if len(trimmed) > 0 {
			result = append(result, trimmed)
		}
	}
	return result
}
