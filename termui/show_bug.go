package termui

import (
	"fmt"
	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/bug/operations"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/util"
	"github.com/jroimartin/gocui"
)

const showBugView = "showBugView"
const showBugSidebarView = "showBugSidebarView"
const showBugInstructionView = "showBugInstructionView"

const timeLayout = "Jan _2 2006"

type showBug struct {
	cache cache.RepoCacher
	bug   *bug.Snapshot
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

		v.Frame = false
	}

	v.Clear()
	sb.renderMain(v)

	v, err = g.SetView(showBugSidebarView, maxX*2/3+1, 0, maxX-1, maxY-2)

	if err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}

		v.Frame = false
	}

	v.Clear()
	sb.renderSidebar(v)

	v, err = g.SetView(showBugInstructionView, -1, maxY-2, maxX, maxY)

	if err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}

		v.Frame = false
		v.BgColor = gocui.ColorBlue

		fmt.Fprintf(v, "[q] Return")
	}

	_, err = g.SetCurrentView(showBugView)
	return err
}

func (sb *showBug) keybindings(g *gocui.Gui) error {
	// Return
	if err := g.SetKeybinding(showBugView, 'q', gocui.ModNone, sb.back); err != nil {
		return err
	}

	if err := g.SetKeybinding(showBugView, gocui.KeyPgup, gocui.ModNone,
		sb.scrollUp); err != nil {
		return err
	}
	if err := g.SetKeybinding(showBugView, gocui.KeyPgdn, gocui.ModNone,
		sb.scrollDown); err != nil {
		return err
	}

	return nil
}

func (sb *showBug) disable(g *gocui.Gui) error {
	if err := g.DeleteView(showBugView); err != nil {
		return err
	}
	if err := g.DeleteView(showBugSidebarView); err != nil {
		return err
	}
	if err := g.DeleteView(showBugInstructionView); err != nil {
		return err
	}
	return nil
}

func (sb *showBug) renderMain(v *gocui.View) {
	maxX, _ := v.Size()

	header1 := fmt.Sprintf("[%s] %s", sb.bug.HumanId(), sb.bug.Title)
	fmt.Fprintf(v, util.LeftPaddedString(header1, maxX, 2)+"\n\n")

	header2 := fmt.Sprintf("[%s] %s opened this bug on %s",
		sb.bug.Status, sb.bug.Author.Name, sb.bug.CreatedAt.Format(timeLayout))
	fmt.Fprintf(v, util.LeftPaddedString(header2, maxX, 2)+"\n\n")

	for _, op := range sb.bug.Operations {
		switch op.(type) {

		case operations.CreateOperation:
			create := op.(operations.CreateOperation)
			fmt.Fprintf(v, util.LeftPaddedString(create.Message, maxX, 6)+"\n\n\n")

		case operations.AddCommentOperation:
			comment := op.(operations.AddCommentOperation)
			header := fmt.Sprintf("%s commented on %s",
				comment.Author.Name, comment.Time().Format(timeLayout))
			fmt.Fprintf(v, util.LeftPaddedString(header, maxX, 6)+"\n\n")
			fmt.Fprintf(v, util.LeftPaddedString(comment.Message, maxX, 6)+"\n\n\n")
		}
	}

}

func (sb *showBug) renderSidebar(v *gocui.View) {
	maxX, _ := v.Size()

	title := util.LeftPaddedString("LABEL", maxX, 2)
	fmt.Fprintf(v, title+"\n\n")

	for _, label := range sb.bug.Labels {
		fmt.Fprintf(v, util.LeftPaddedString(label.String(), maxX, 2))
		fmt.Fprintln(v)
	}
}

func (sb *showBug) back(g *gocui.Gui, v *gocui.View) error {
	sb.bug = nil
	ui.activateWindow(ui.bugTable)
	return nil
}

func (sb *showBug) scrollUp(g *gocui.Gui, v *gocui.View) error {
	return nil
}

func (sb *showBug) scrollDown(g *gocui.Gui, v *gocui.View) error {
	return nil
}
