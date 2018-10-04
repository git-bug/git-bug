package termui
import (
	"fmt"
	"strings"
	"github.com/jroimartin/gocui"
	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/cache"
)
const labelSelectView = "labelSelectView"
const labelSelectInstructionsView = "labelSelectInstructionsView"

type labelSelect struct {
	cache       *cache.RepoCache
	bug         *cache.BugCache
	labels      []bug.Label
	labelSelect []bool
	selected    int
	childViews  []string
}

func newLabelSelect() *labelSelect {
	return &labelSelect{}
}

func (ls *labelSelect) SetBug(cache *cache.RepoCache, bug *cache.BugCache) {
	ls.cache = cache
	ls.bug = bug
	ls.labels = cache.ValidLabels()

	// Find which labels are currently applied to the bug
	bugLabels := bug.Snapshot().Labels
	labelSelect := make([]bool, len(ls.labels))
	for i, label := range ls.labels {
		for _, bugLabel := range bugLabels {
			if label == bugLabel {
				labelSelect[i] = true
				break
			}
		}
	}

	ls.labelSelect = labelSelect
	ls.selected = 0
}

func (ls *labelSelect) keybindings(g *gocui.Gui) error {
	// Abort
	if err := g.SetKeybinding(labelSelectView, gocui.KeyEsc, gocui.ModNone, ls.abort); err != nil {
		return err
	}
	// Save and return
	if err := g.SetKeybinding(labelSelectView, 'q', gocui.ModNone, ls.saveAndReturn); err != nil {
		return err
	}
	// Up
	if err := g.SetKeybinding(labelSelectView, gocui.KeyArrowUp, gocui.ModNone, ls.selectPrevious); err != nil {
		return err
	}
	if err := g.SetKeybinding(labelSelectView, 'k', gocui.ModNone, ls.selectPrevious); err != nil {
		return err
	}
	// Down
	if err := g.SetKeybinding(labelSelectView, gocui.KeyArrowDown, gocui.ModNone, ls.selectNext); err != nil {
		return err
	}
	if err := g.SetKeybinding(labelSelectView, 'j', gocui.ModNone, ls.selectNext); err != nil {
		return err
	}
	// Select
	if err := g.SetKeybinding(labelSelectView, gocui.KeySpace, gocui.ModNone, ls.selectItem); err != nil {
		return err
	}
	if err := g.SetKeybinding(labelSelectView, 'x', gocui.ModNone, ls.selectItem); err != nil {
		return err
	}
	if err := g.SetKeybinding(labelSelectView, gocui.KeyEnter, gocui.ModNone, ls.selectItem); err != nil {
		return err
	}
	// Add
	if err := g.SetKeybinding(labelSelectView, 'a', gocui.ModNone, ls.addItem); err != nil {
		return err
	}
	return nil
}

func (ls *labelSelect) layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()
	ls.childViews = nil

	// TODO: Make width adaptive
	width := 30
	height := 2*len(ls.labels) + 3
	x0 := 2
	y0 := 2

	v, err := g.SetView(labelSelectView, x0, y0, x0+width, y0+height)
	if err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
	
		v.Frame = false
	}
	y0 += 1

	for i, label := range ls.labels {
		viewname := fmt.Sprintf("view%d", i)
		v, err := g.SetView(viewname, x0+2, y0, x0+width-2, y0+2)
		if err != nil && err != gocui.ErrUnknownView {
			return err
		}
		ls.childViews = append(ls.childViews, viewname)
		v.Frame = i == ls.selected
		v.Clear()
		selectBox := " [ ] "
		if ls.labelSelect[i] {
			selectBox = " [x] "
		}
		fmt.Fprint(v, selectBox, label)
		y0 += 2
	}

	v, err = g.SetView(labelSelectInstructionsView, -1, maxY-2, maxX, maxY)
	ls.childViews = append(ls.childViews, labelSelectInstructionsView)
	if err != nil {
		if err != gocui.ErrUnknownView{
			return err
		}
		v.Frame = false
		v.BgColor = gocui.ColorBlue
	}
	v.Clear()
	fmt.Fprint(v, "[↓↑,jk] Nav [a] Add item [q] Save and close")
	if _, err = g.SetViewOnTop(labelSelectInstructionsView); err != nil {
		return err
	}
	if _, err := g.SetCurrentView(labelSelectView); err != nil {
		return err
	}
	return nil
}

func (ls *labelSelect) disable(g *gocui.Gui) error {
	for _, view := range ls.childViews {
		if err := g.DeleteView(view); err != nil && err != gocui.ErrUnknownView {
			return err
		}
	}
	return nil
}

func(ls *labelSelect) selectPrevious(g *gocui.Gui, v*gocui.View) error {
	ls.selected = maxInt(0, ls.selected-1)
	return nil
}

func(ls *labelSelect) selectNext(g *gocui.Gui, v*gocui.View) error {
	ls.selected = minInt(len(ls.labels)-1, ls.selected+1)
	return nil
}

func(ls *labelSelect) selectItem(g *gocui.Gui, v*gocui.View) error {
	ls.labelSelect[ls.selected] = !ls.labelSelect[ls.selected]
	return nil
}

func (ls *labelSelect) addItem(g *gocui.Gui, v *gocui.View) error {
	c := ui.inputPopup.Activate("Add a new label")

	go func() {
		input := <-c
		
		// Standardize label format
		input = strings.TrimSuffix(input, "\n")
		input = strings.Replace(input, " ", "-", -1)

		// Check if label already exists
		for i, label := range ls.labels {
			if input == label.String() {
				ls.labelSelect[i] = true
				ls.selected = i
				return
			}
		}

		// Add new label, make it selected, and move frame
		ls.labels = append(ls.labels, bug.Label(input))
		ls.labelSelect = append(ls.labelSelect, true)
		ls.selected = len(ls.labels) - 1
	}()
	return nil
}

func (ls *labelSelect) abort(g *gocui.Gui, v *gocui.View) error {
	return ui.activateWindow(ui.showBug)
}

func (ls *labelSelect) saveAndReturn(g *gocui.Gui, v *gocui.View) error {
	bugLabels := ls.bug.Snapshot().Labels
	selectedLabels := []bug.Label{}
	for i, label := range ls.labels {
		if ls.labelSelect[i] {
			selectedLabels = append(selectedLabels, label)
		}
	}

	// Find the new and removed labels. This makes use of the fact that the first elements
	// of selectedLabels are the not-removed labels in bugLabels
	newLabels := []string{}
	rmLabels := []string{}
	i := 0	// Index for bugLabels
	j := 0	// Index for selectedLabels
	for {
		if j == len(selectedLabels) {
			// No more labels to consider
			break
		} else if i == len(bugLabels) {
			// Remaining labels are all new
			newLabels = append(newLabels, selectedLabels[j].String())
			j += 1
		} else if bugLabels[i] == selectedLabels[j] {
			// Labels match. Move to next pair
			i += 1
			j += 1
		} else {
			// Labels don't match. Prelabel must have been removed
			rmLabels = append(rmLabels, bugLabels[i].String())
			i += 1
		}
	}

	if _, err := ls.bug.ChangeLabels(newLabels, rmLabels); err != nil {
		ui.msgPopup.Activate(msgPopupErrorTitle, err.Error())
	}

	return ui.activateWindow(ui.showBug)
}

// func (ls *labelSelect) Activate(labels []bug.Label, sel []bool) <-chan []bug.Label {
// 	ls.labels = labels
// 	ls.labelSelect = sel
// 	ls.selected = 0
// 	ls.c = make(chan []bug.Label)
// 	return ls.c
// }