package config

import (
	"strings"

	"code.rocketnine.space/tslocum/cview"
	"github.com/gdamore/tcell/v2"
	"github.com/spf13/viper"
)

// NOTE: CmdLink[1-90] and CmdTab[1-90] need to be in-order and consecutive
// This property is used to simplify key handling in display/display.go
type Command int

const (
	CmdInvalid Command = 0
	CmdLink1           = 1
	CmdLink2           = 2
	CmdLink3           = 3
	CmdLink4           = 4
	CmdLink5           = 5
	CmdLink6           = 6
	CmdLink7           = 7
	CmdLink8           = 8
	CmdLink9           = 9
	CmdLink0           = 10
	CmdTab1            = 11
	CmdTab2            = 12
	CmdTab3            = 13
	CmdTab4            = 14
	CmdTab5            = 15
	CmdTab6            = 16
	CmdTab7            = 17
	CmdTab8            = 18
	CmdTab9            = 19
	CmdTab0            = 20
	CmdBottom          = iota
	CmdEdit
	CmdHome
	CmdBookmarks
	CmdAddBookmark
	CmdSave
	CmdReload
	CmdBack
	CmdForward
	CmdMoveUp
	CmdMoveDown
	CmdMoveLeft
	CmdMoveRight
	CmdPgup
	CmdPgdn
	CmdNewTab
	CmdCloseTab
	CmdNextTab
	CmdPrevTab
	CmdQuit
	CmdHelp
	CmdSub
	CmdAddSub
	CmdCopyPageURL
	CmdCopyTargetURL
	CmdBeginning
	CmdEnd
	CmdSearch
	CmdNextMatch
	CmdPrevMatch
)

type keyBinding struct {
	key tcell.Key
	mod tcell.ModMask
	r   rune
}

// Map of active keybindings to commands.
var bindings map[keyBinding]Command

// inversion of tcell.KeyNames, used to simplify config parsing.
// used by parseBinding() below.
var tcellKeys map[string]tcell.Key

// helper function that takes a single keyBinding object and returns
// a string in the format used by the configuration file.  Support
// function for GetKeyBinding(), used to make the help panel helpful.
func keyBindingToString(kb keyBinding) (string, bool) {
	var prefix string = ""

	if kb.mod&tcell.ModAlt == tcell.ModAlt {
		prefix = "Alt-"
	}

	if kb.key == tcell.KeyRune {
		if kb.r == ' ' {
			return prefix + "Space", true
		}
		return prefix + string(kb.r), true
	}
	s, ok := tcell.KeyNames[kb.key]
	if ok {
		return prefix + s, true
	}
	return "", false
}

// Get all keybindings for a Command as a string.
// Used by the help panel so bindable keys display with their
// bound values rather than hardcoded defaults.
func GetKeyBinding(cmd Command) string {
	var s string = ""
	for kb, c := range bindings {
		if c == cmd {
			t, ok := keyBindingToString(kb)
			if ok {
				s += t + ", "
			}
		}
	}

	if len(s) > 0 {
		return s[:len(s)-2]
	}
	return s
}

// Parse a single keybinding string and add it to the binding map
func parseBinding(cmd Command, binding string) {
	var k tcell.Key
	var m tcell.ModMask = 0
	var r rune = 0

	if strings.HasPrefix(binding, "Alt-") {
		m = tcell.ModAlt
		binding = binding[4:]
	}

	if len([]rune(binding)) == 1 {
		k = tcell.KeyRune
		r = []rune(binding)[0]
	} else if len(binding) == 0 {
		return
	} else if binding == "Space" {
		k = tcell.KeyRune
		r = ' '
	} else {
		var ok bool
		k, ok = tcellKeys[binding]
		if !ok { // Bad keybinding!  Quietly ignore...
			return
		}
		if strings.HasPrefix(binding, "Ctrl") {
			m += tcell.ModCtrl
		}
	}

	bindings[keyBinding{k, m, r}] = cmd
}

// Generate the bindings map from the TOML configuration file.
// Called by config.Init()
func KeyInit() {
	configBindings := map[Command]string{
		CmdLink1:         "keybindings.bind_link1",
		CmdLink2:         "keybindings.bind_link2",
		CmdLink3:         "keybindings.bind_link3",
		CmdLink4:         "keybindings.bind_link4",
		CmdLink5:         "keybindings.bind_link5",
		CmdLink6:         "keybindings.bind_link6",
		CmdLink7:         "keybindings.bind_link7",
		CmdLink8:         "keybindings.bind_link8",
		CmdLink9:         "keybindings.bind_link9",
		CmdLink0:         "keybindings.bind_link0",
		CmdBottom:        "keybindings.bind_bottom",
		CmdEdit:          "keybindings.bind_edit",
		CmdHome:          "keybindings.bind_home",
		CmdBookmarks:     "keybindings.bind_bookmarks",
		CmdAddBookmark:   "keybindings.bind_add_bookmark",
		CmdSave:          "keybindings.bind_save",
		CmdReload:        "keybindings.bind_reload",
		CmdBack:          "keybindings.bind_back",
		CmdForward:       "keybindings.bind_forward",
		CmdMoveUp:        "keybindings.bind_moveup",
		CmdMoveDown:      "keybindings.bind_movedown",
		CmdMoveLeft:      "keybindings.bind_moveleft",
		CmdMoveRight:     "keybindings.bind_moveright",
		CmdPgup:          "keybindings.bind_pgup",
		CmdPgdn:          "keybindings.bind_pgdn",
		CmdNewTab:        "keybindings.bind_new_tab",
		CmdCloseTab:      "keybindings.bind_close_tab",
		CmdNextTab:       "keybindings.bind_next_tab",
		CmdPrevTab:       "keybindings.bind_prev_tab",
		CmdQuit:          "keybindings.bind_quit",
		CmdHelp:          "keybindings.bind_help",
		CmdSub:           "keybindings.bind_sub",
		CmdAddSub:        "keybindings.bind_add_sub",
		CmdCopyPageURL:   "keybindings.bind_copy_page_url",
		CmdCopyTargetURL: "keybindings.bind_copy_target_url",
		CmdBeginning:     "keybindings.bind_beginning",
		CmdEnd:           "keybindings.bind_end",
		CmdSearch:		  "keybindings.bind_search",
		CmdNextMatch:	  "keybindings.bind_next_match",
		CmdPrevMatch:	  "keybindings.bind_prev_match",
	}
	// This is split off to allow shift_numbers to override bind_tab[1-90]
	// (This is needed for older configs so that the default bind_tab values
	// aren't used)
	configTabNBindings := map[Command]string{
		CmdTab1: "keybindings.bind_tab1",
		CmdTab2: "keybindings.bind_tab2",
		CmdTab3: "keybindings.bind_tab3",
		CmdTab4: "keybindings.bind_tab4",
		CmdTab5: "keybindings.bind_tab5",
		CmdTab6: "keybindings.bind_tab6",
		CmdTab7: "keybindings.bind_tab7",
		CmdTab8: "keybindings.bind_tab8",
		CmdTab9: "keybindings.bind_tab9",
		CmdTab0: "keybindings.bind_tab0",
	}
	tcellKeys = make(map[string]tcell.Key)
	bindings = make(map[keyBinding]Command)

	for k, kname := range tcell.KeyNames {
		tcellKeys[kname] = k
	}

	// Set cview navigation keys to use user-set ones
	cview.Keys.MoveUp2 = viper.GetStringSlice(configBindings[CmdMoveUp])
	cview.Keys.MoveDown2 = viper.GetStringSlice(configBindings[CmdMoveDown])
	cview.Keys.MoveLeft2 = viper.GetStringSlice(configBindings[CmdMoveLeft])
	cview.Keys.MoveRight2 = viper.GetStringSlice(configBindings[CmdMoveRight])
	cview.Keys.MoveFirst = viper.GetStringSlice(configBindings[CmdBeginning])
	cview.Keys.MoveFirst2 = nil
	cview.Keys.MoveLast = viper.GetStringSlice(configBindings[CmdEnd])
	cview.Keys.MoveLast2 = nil

	for c, allb := range configBindings {
		for _, b := range viper.GetStringSlice(allb) {
			parseBinding(c, b)
		}
	}

	// Backwards compatibility with the old shift_numbers config line.
	shiftNumbers := []rune(viper.GetString("keybindings.shift_numbers"))
	if len(shiftNumbers) > 0 && len(shiftNumbers) <= 10 {
		for i, r := range shiftNumbers {
			bindings[keyBinding{tcell.KeyRune, 0, r}] = CmdTab1 + Command(i)
		}
	} else {
		for c, allb := range configTabNBindings {
			for _, b := range viper.GetStringSlice(allb) {
				parseBinding(c, b)
			}
		}
	}
}

// Used by the display package to turn a tcell.EventKey into a Command
func TranslateKeyEvent(e *tcell.EventKey) Command {
	var ok bool
	var cmd Command
	k := e.Key()
	if k == tcell.KeyRune {
		cmd, ok = bindings[keyBinding{k, e.Modifiers(), e.Rune()}]
	} else { // Sometimes tcell sets e.Rune() on non-KeyRune events.
		cmd, ok = bindings[keyBinding{k, e.Modifiers(), 0}]
	}
	if ok {
		return cmd
	}
	return CmdInvalid
}
