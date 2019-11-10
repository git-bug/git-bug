// Copyright 2014 The gocui Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gocui

import (
	"strings"

	"github.com/awesome-gocui/termbox-go"
)

// Key represents special keys or keys combinations.
type Key termbox.Key

// Modifier allows to define special keys combinations. They can be used
// in combination with Keys or Runes when a new keybinding is defined.
type Modifier termbox.Modifier

// Keybidings are used to link a given key-press event with a handler.
type keybinding struct {
	viewName string
	key      Key
	ch       rune
	mod      Modifier
	handler  func(*Gui, *View) error
}

// Parse takes the input string and extracts the keybinding.
// Returns a Key / rune, a Modifier and an error.
func Parse(input string) (interface{}, Modifier, error) {
	if len(input) == 1 {
		_, r, err := getKey(rune(input[0]))
		if err != nil {
			return nil, ModNone, err
		}
		return r, ModNone, nil
	}

	var modifier Modifier
	cleaned := make([]string, 0)

	tokens := strings.Split(input, "+")
	for _, t := range tokens {
		normalized := strings.Title(strings.ToLower(t))
		if t == "Alt" {
			modifier = ModAlt
			continue
		}
		cleaned = append(cleaned, normalized)
	}

	key, exist := translate[strings.Join(cleaned, "")]
	if !exist {
		return nil, ModNone, ErrNoSuchKeybind
	}

	return key, modifier, nil
}

// ParseAll takes an array of strings and returns a map of all keybindings.
func ParseAll(input []string) (map[interface{}]Modifier, error) {
	ret := make(map[interface{}]Modifier)
	for _, i := range input {
		k, m, err := Parse(i)
		if err != nil {
			return ret, err
		}
		ret[k] = m
	}
	return ret, nil
}

// MustParse takes the input string and returns a Key / rune and a Modifier.
// It will panic if any error occured.
func MustParse(input string) (interface{}, Modifier) {
	k, m, err := Parse(input)
	if err != nil {
		panic(err)
	}
	return k, m
}

// MustParseAll takes an array of strings and returns a map of all keybindings.
// It will panic if any error occured.
func MustParseAll(input []string) map[interface{}]Modifier {
	result, err := ParseAll(input)
	if err != nil {
		panic(err)
	}
	return result
}

// newKeybinding returns a new Keybinding object.
func newKeybinding(viewname string, key Key, ch rune, mod Modifier, handler func(*Gui, *View) error) (kb *keybinding) {
	kb = &keybinding{
		viewName: viewname,
		key:      key,
		ch:       ch,
		mod:      mod,
		handler:  handler,
	}
	return kb
}

// matchKeypress returns if the keybinding matches the keypress.
func (kb *keybinding) matchKeypress(key Key, ch rune, mod Modifier) bool {
	return kb.key == key && kb.ch == ch && kb.mod == mod
}

// matchView returns if the keybinding matches the current view.
func (kb *keybinding) matchView(v *View) bool {
	// if the user is typing in a field, ignore char keys
	if v == nil || (v.Editable && kb.ch != 0) {
		return false
	}
	return kb.viewName == v.name
}

// translations for strings to keys
var translate = map[string]Key{
	"F1":             KeyF1,
	"F2":             KeyF2,
	"F3":             KeyF3,
	"F4":             KeyF4,
	"F5":             KeyF5,
	"F6":             KeyF6,
	"F7":             KeyF7,
	"F8":             KeyF8,
	"F9":             KeyF9,
	"F10":            KeyF10,
	"F11":            KeyF11,
	"F12":            KeyF12,
	"Insert":         KeyInsert,
	"Delete":         KeyDelete,
	"Home":           KeyHome,
	"End":            KeyEnd,
	"Pgup":           KeyPgup,
	"Pgdn":           KeyPgdn,
	"ArrowUp":        KeyArrowUp,
	"ArrowDown":      KeyArrowDown,
	"ArrowLeft":      KeyArrowLeft,
	"ArrowRight":     KeyArrowRight,
	"CtrlTilde":      KeyCtrlTilde,
	"Ctrl2":          KeyCtrl2,
	"CtrlSpace":      KeyCtrlSpace,
	"CtrlA":          KeyCtrlA,
	"CtrlB":          KeyCtrlB,
	"CtrlC":          KeyCtrlC,
	"CtrlD":          KeyCtrlD,
	"CtrlE":          KeyCtrlE,
	"CtrlF":          KeyCtrlF,
	"CtrlG":          KeyCtrlG,
	"Backspace":      KeyBackspace,
	"CtrlH":          KeyCtrlH,
	"Tab":            KeyTab,
	"CtrlI":          KeyCtrlI,
	"CtrlJ":          KeyCtrlJ,
	"CtrlK":          KeyCtrlK,
	"CtrlL":          KeyCtrlL,
	"Enter":          KeyEnter,
	"CtrlM":          KeyCtrlM,
	"CtrlN":          KeyCtrlN,
	"CtrlO":          KeyCtrlO,
	"CtrlP":          KeyCtrlP,
	"CtrlQ":          KeyCtrlQ,
	"CtrlR":          KeyCtrlR,
	"CtrlS":          KeyCtrlS,
	"CtrlT":          KeyCtrlT,
	"CtrlU":          KeyCtrlU,
	"CtrlV":          KeyCtrlV,
	"CtrlW":          KeyCtrlW,
	"CtrlX":          KeyCtrlX,
	"CtrlY":          KeyCtrlY,
	"CtrlZ":          KeyCtrlZ,
	"Esc":            KeyEsc,
	"CtrlLsqBracket": KeyCtrlLsqBracket,
	"Ctrl3":          KeyCtrl3,
	"Ctrl4":          KeyCtrl4,
	"CtrlBackslash":  KeyCtrlBackslash,
	"Ctrl5":          KeyCtrl5,
	"CtrlRsqBracket": KeyCtrlRsqBracket,
	"Ctrl6":          KeyCtrl6,
	"Ctrl7":          KeyCtrl7,
	"CtrlSlash":      KeyCtrlSlash,
	"CtrlUnderscore": KeyCtrlUnderscore,
	"Space":          KeySpace,
	"Backspace2":     KeyBackspace2,
	"Ctrl8":          KeyCtrl8,
	"Mouseleft":      MouseLeft,
	"Mousemiddle":    MouseMiddle,
	"Mouseright":     MouseRight,
	"Mouserelease":   MouseRelease,
	"MousewheelUp":   MouseWheelUp,
	"MousewheelDown": MouseWheelDown,
}

// Special keys.
const (
	KeyF1         Key = Key(termbox.KeyF1)
	KeyF2             = Key(termbox.KeyF2)
	KeyF3             = Key(termbox.KeyF3)
	KeyF4             = Key(termbox.KeyF4)
	KeyF5             = Key(termbox.KeyF5)
	KeyF6             = Key(termbox.KeyF6)
	KeyF7             = Key(termbox.KeyF7)
	KeyF8             = Key(termbox.KeyF8)
	KeyF9             = Key(termbox.KeyF9)
	KeyF10            = Key(termbox.KeyF10)
	KeyF11            = Key(termbox.KeyF11)
	KeyF12            = Key(termbox.KeyF12)
	KeyInsert         = Key(termbox.KeyInsert)
	KeyDelete         = Key(termbox.KeyDelete)
	KeyHome           = Key(termbox.KeyHome)
	KeyEnd            = Key(termbox.KeyEnd)
	KeyPgup           = Key(termbox.KeyPgup)
	KeyPgdn           = Key(termbox.KeyPgdn)
	KeyArrowUp        = Key(termbox.KeyArrowUp)
	KeyArrowDown      = Key(termbox.KeyArrowDown)
	KeyArrowLeft      = Key(termbox.KeyArrowLeft)
	KeyArrowRight     = Key(termbox.KeyArrowRight)

	MouseLeft      = Key(termbox.MouseLeft)
	MouseMiddle    = Key(termbox.MouseMiddle)
	MouseRight     = Key(termbox.MouseRight)
	MouseRelease   = Key(termbox.MouseRelease)
	MouseWheelUp   = Key(termbox.MouseWheelUp)
	MouseWheelDown = Key(termbox.MouseWheelDown)
)

// Keys combinations.
const (
	KeyCtrlTilde      Key = Key(termbox.KeyCtrlTilde)
	KeyCtrl2              = Key(termbox.KeyCtrl2)
	KeyCtrlSpace          = Key(termbox.KeyCtrlSpace)
	KeyCtrlA              = Key(termbox.KeyCtrlA)
	KeyCtrlB              = Key(termbox.KeyCtrlB)
	KeyCtrlC              = Key(termbox.KeyCtrlC)
	KeyCtrlD              = Key(termbox.KeyCtrlD)
	KeyCtrlE              = Key(termbox.KeyCtrlE)
	KeyCtrlF              = Key(termbox.KeyCtrlF)
	KeyCtrlG              = Key(termbox.KeyCtrlG)
	KeyBackspace          = Key(termbox.KeyBackspace)
	KeyCtrlH              = Key(termbox.KeyCtrlH)
	KeyTab                = Key(termbox.KeyTab)
	KeyCtrlI              = Key(termbox.KeyCtrlI)
	KeyCtrlJ              = Key(termbox.KeyCtrlJ)
	KeyCtrlK              = Key(termbox.KeyCtrlK)
	KeyCtrlL              = Key(termbox.KeyCtrlL)
	KeyEnter              = Key(termbox.KeyEnter)
	KeyCtrlM              = Key(termbox.KeyCtrlM)
	KeyCtrlN              = Key(termbox.KeyCtrlN)
	KeyCtrlO              = Key(termbox.KeyCtrlO)
	KeyCtrlP              = Key(termbox.KeyCtrlP)
	KeyCtrlQ              = Key(termbox.KeyCtrlQ)
	KeyCtrlR              = Key(termbox.KeyCtrlR)
	KeyCtrlS              = Key(termbox.KeyCtrlS)
	KeyCtrlT              = Key(termbox.KeyCtrlT)
	KeyCtrlU              = Key(termbox.KeyCtrlU)
	KeyCtrlV              = Key(termbox.KeyCtrlV)
	KeyCtrlW              = Key(termbox.KeyCtrlW)
	KeyCtrlX              = Key(termbox.KeyCtrlX)
	KeyCtrlY              = Key(termbox.KeyCtrlY)
	KeyCtrlZ              = Key(termbox.KeyCtrlZ)
	KeyEsc                = Key(termbox.KeyEsc)
	KeyCtrlLsqBracket     = Key(termbox.KeyCtrlLsqBracket)
	KeyCtrl3              = Key(termbox.KeyCtrl3)
	KeyCtrl4              = Key(termbox.KeyCtrl4)
	KeyCtrlBackslash      = Key(termbox.KeyCtrlBackslash)
	KeyCtrl5              = Key(termbox.KeyCtrl5)
	KeyCtrlRsqBracket     = Key(termbox.KeyCtrlRsqBracket)
	KeyCtrl6              = Key(termbox.KeyCtrl6)
	KeyCtrl7              = Key(termbox.KeyCtrl7)
	KeyCtrlSlash          = Key(termbox.KeyCtrlSlash)
	KeyCtrlUnderscore     = Key(termbox.KeyCtrlUnderscore)
	KeySpace              = Key(termbox.KeySpace)
	KeyBackspace2         = Key(termbox.KeyBackspace2)
	KeyCtrl8              = Key(termbox.KeyCtrl8)
)

// Modifiers.
const (
	ModNone Modifier = Modifier(0)
	ModAlt           = Modifier(termbox.ModAlt)
)
