package display

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"code.rocketnine.space/tslocum/cview"
	"github.com/gdamore/tcell/v2"
	"github.com/makeworld-the-better-one/amfora/cache"
	"github.com/makeworld-the-better-one/amfora/config"
	"github.com/makeworld-the-better-one/amfora/renderer"
	"github.com/makeworld-the-better-one/amfora/structs"
	"github.com/makeworld-the-better-one/go-gemini"
	"github.com/spf13/viper"
)

var tabs []*tab // Slice of all the current browser tabs
var curTab = -1 // What tab is currently visible - index for the tabs slice (-1 means there are no tabs)

// Terminal dimensions
var termW int
var termH int

// The user input and URL display bar at the bottom
var bottomBar = cview.NewInputField()

// When the bottom bar string has a space, this regex decides whether it's
// a non-encoded URL or a search string.
// See this comment for details:
// https://github.com/makeworld-the-better-one/amfora/issues/138#issuecomment-740961292
var hasSpaceisURL = regexp.MustCompile(`[^ ]+\.[^ ].*/.`)

// Viewer for the tab primitives
// Pages are named as strings of tab numbers - so the textview for the first tab
// is held in the page named "0".
// The only pages that don't confine to this scheme are those named after modals,
// which are used to draw modals on top the current tab.
// Ex: "info", "error", "input", "yesno"
var panels = cview.NewPanels()

// Tabbed viewer for primitives
// Panels are named as strings of tab numbers - so the textview for the first tab
// is held in the page named "0".
var browser = cview.NewTabbedPanels()

// Root layout
var layout = cview.NewFlex()

var newTabPage structs.Page

// Global mutex for changing the size of the left margin on all tabs.
var reformatMu = sync.Mutex{}

var App = cview.NewApplication()

func Init(version, commit, builtBy string) {
	aboutInit(version, commit, builtBy)

	App.EnableMouse(false)
	App.SetRoot(layout, true)
	App.SetAfterResizeFunc(func(width int, height int) {
		// Store for calculations
		termW = width
		termH = height

		// Make sure the current tab content is reformatted when the terminal size changes
		go func(t *tab) {
			reformatMu.Lock() // Only allow one reformat job at a time
			for i := range tabs {
				// Overwrite all tabs with a new, differently sized, left margin
				browser.AddTab(
					strconv.Itoa(i),
					makeTabLabel(strconv.Itoa(i+1)),
					makeContentLayout(tabs[i].view, leftMargin()),
				)
				if tabs[i] == t {
					// Reformat page ASAP, in the middle of loop
					reformatPageAndSetView(t, t.page)
				}
			}
			App.Draw()
			reformatMu.Unlock()
		}(tabs[curTab])
	})

	panels.AddPanel("browser", browser, true, true)

	helpInit()

	layout.SetDirection(cview.FlexRow)
	layout.AddItem(panels, 0, 1, true)
	layout.AddItem(bottomBar, 1, 1, false)

	if viper.GetBool("a-general.color") {
		layout.SetBackgroundColor(config.GetColor("bg"))

		bottomBar.SetBackgroundColor(config.GetColor("bottombar_bg"))
		bottomBar.SetLabelColor(config.GetColor("bottombar_label"))
		bottomBar.SetFieldBackgroundColor(config.GetColor("bottombar_bg"))
		bottomBar.SetFieldTextColor(config.GetColor("bottombar_text"))

		browser.SetTabBackgroundColor(config.GetColor("bg"))
		browser.SetTabBackgroundColorFocused(config.GetColor("tab_num"))
		browser.SetTabTextColor(config.GetColor("tab_num"))
		browser.SetTabTextColorFocused(config.GetTextColor("bg", "tab_num"))
		browser.SetTabSwitcherDivider(
			"",
			fmt.Sprintf("[%s:%s]|[-]", config.GetColorString("tab_divider"), config.GetColorString("bg")),
			fmt.Sprintf("[%s:%s]|[-]", config.GetColorString("tab_divider"), config.GetColorString("bg")),
		)
		browser.Switcher.SetBackgroundColor(config.GetColor("bg"))
	} else {
		bottomBar.SetBackgroundColor(tcell.ColorWhite)
		bottomBar.SetLabelColor(tcell.ColorBlack)
		bottomBar.SetFieldBackgroundColor(tcell.ColorWhite)
		bottomBar.SetFieldTextColor(tcell.ColorBlack)

		browser.SetTabBackgroundColor(tcell.ColorBlack)
		browser.SetTabBackgroundColorFocused(tcell.ColorWhite)
		browser.SetTabTextColor(tcell.ColorWhite)
		browser.SetTabTextColorFocused(tcell.ColorBlack)
		browser.SetTabSwitcherDivider(
			"",
			"[#ffffff:#000000]|[-]",
			"[#ffffff:#000000]|[-]",
		)
	}

	bottomBar.SetDoneFunc(func(key tcell.Key) {
		tab := curTab

		// Reset func to set the bottomBar back to what it was before
		// Use for errors.
		reset := func() {
			bottomBar.SetLabel("")
			tabs[tab].applyAll()
			App.SetFocus(tabs[tab].view)
		}

		//nolint:exhaustive
		switch key {
		case tcell.KeyEnter:
			// Figure out whether it's a URL, link number, or search
			// And send out a request

			query := bottomBar.GetText()

			if strings.TrimSpace(query) == "" {
				// Ignore
				reset()
				return
			}
			if query[0] == '.' && tabs[tab].hasContent() && !tabs[tab].isAnAboutPage() {
				// Relative url
				current, err := url.Parse(tabs[tab].page.URL)
				if err != nil {
					// This shouldn't occur
					return
				}

				if query == ".." && tabs[tab].page.URL[len(tabs[tab].page.URL)-1] != '/' {
					// Support what ".." used to work like
					// If on /dir/doc.gmi, got to /dir/
					query = "./"
				}

				target, err := current.Parse(query)
				if err != nil {
					// Invalid relative url
					return
				}
				URL(target.String())
				return
			}

			i, err := strconv.Atoi(query)
			if err != nil {
				if strings.HasPrefix(query, "new:") && len(query) > 4 {
					// They're trying to open a link number in a new tab
					i, err = strconv.Atoi(query[4:])
					if err != nil {
						reset()
						return
					}
					if i <= len(tabs[tab].page.Links) && i > 0 {
						// Open new tab and load link
						oldTab := tab
						NewTab()
						// Resolve and follow link manually
						prevParsed, _ := url.Parse(tabs[oldTab].page.URL)
						nextParsed, err := url.Parse(tabs[oldTab].page.Links[i-1])
						if err != nil {
							Error("URL Error", "link URL could not be parsed")
							reset()
							return
						}
						URL(prevParsed.ResolveReference(nextParsed).String())
						return
					}
				} else {
					// It's a full URL or search term
					// Detect if it's a search or URL

					// Remove whitespace from the string.
					// We don't want to convert legitimate
					// :// links to search terms.
					query := strings.TrimSpace(query)
					if (strings.Contains(query, " ") && !hasSpaceisURL.MatchString(query)) ||
						(!strings.HasPrefix(query, "//") && !strings.Contains(query, "://") &&
							!strings.Contains(query, ".")) && !strings.HasPrefix(query, "about:") {
						// Has a space and follows regex, OR
						// doesn't start with "//", contain "://", and doesn't have a dot either.
						// Then it's a search

						u := viper.GetString("a-general.search") + "?" + gemini.QueryEscape(query)
						// Don't use the cached version of the search
						cache.RemovePage(normalizeURL(u))
						URL(u)
					} else {
						// Full URL
						// Don't use cached version for manually entered URL
						cache.RemovePage(normalizeURL(fixUserURL(query)))
						URL(query)
					}
					return
				}
			}
			if i <= len(tabs[tab].page.Links) && i > 0 {
				// It's a valid link number
				followLink(tabs[tab], tabs[tab].page.URL, tabs[tab].page.Links[i-1])
				return
			}
			// Invalid link number, don't do anything
			reset()
			return

		case tcell.KeyEsc:
			// Set back to what it was
			reset()
			return
		}
		// Other potential keys are Tab and Backtab, they are ignored
	})

	// Render the default new tab content ONCE and store it for later
	// This code is repeated in Reload()
	newTabContent := getNewTabContent()
	renderedNewTabContent, newTabLinks := renderer.RenderGemini(newTabContent, textWidth(), false)
	newTabPage = structs.Page{
		Raw:       newTabContent,
		Content:   renderedNewTabContent,
		Links:     newTabLinks,
		URL:       "about:newtab",
		TermWidth: -1, // Force reformatting on first display
		Mediatype: structs.TextGemini,
	}

	modalInit()

	// Setup map of keys to functions here
	// Changing tabs, new tab, etc
	App.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		_, ok := App.GetFocus().(*cview.Button)
		if ok {
			// It's focused on a modal right now, nothing should interrupt
			return event
		}
		_, ok = App.GetFocus().(*cview.InputField)
		if ok {
			// An InputField is in focus, nothing should interrupt
			return event
		}
		_, ok = App.GetFocus().(*cview.Modal)
		if ok {
			// It's focused on a modal right now, nothing should interrupt
			return event
		}
		_, ok = App.GetFocus().(*cview.Table)
		if ok {
			// It's focused on help right now
			return event
		}

		// To add a configurable global key command, you'll need to update one of
		// the two switch statements here.  You'll also need to add an enum entry in
		// config/keybindings.go, update KeyInit() in config/keybindings.go, add a default
		// keybinding in config/config.go and update the help panel in display/help.go

		cmd := config.TranslateKeyEvent(event)
		if tabs[curTab].mode == tabModeDone {
			// All the keys and operations that can only work while NOT loading
			//nolint:exhaustive
			switch cmd {
			case config.CmdReload:
				Reload()
				return nil
			case config.CmdHome:
				URL(viper.GetString("a-general.home"))
				return nil
			case config.CmdBottom:
				// Space starts typing, like Bombadillo
				bottomBar.SetLabel("[::b]URL/Num./Search: [::-]")
				bottomBar.SetText("")
				// Don't save bottom bar, so that whenever you switch tabs, it's not in that mode
				App.SetFocus(bottomBar)
				return nil
			case config.CmdEdit:
				// Letter e allows to edit current URL
				bottomBar.SetLabel("[::b]Edit URL: [::-]")
				bottomBar.SetText(tabs[curTab].page.URL)
				App.SetFocus(bottomBar)
				return nil
			case config.CmdAddSub:
				go addSubscription()
				return nil
			}
		}

		// All the keys and operations that can work while a tab IS loading
		//nolint:exhaustive
		switch cmd {
		case config.CmdNewTab:
			if tabs[curTab].page.Mode == structs.ModeLinkSelect {
				next, err := resolveRelLink(tabs[curTab], tabs[curTab].page.URL, tabs[curTab].page.Selected)
				if err != nil {
					Error("URL Error", err.Error())
					return nil
				}
				NewTab()
				URL(next)
			} else {
				NewTab()
			}
			return nil
		case config.CmdCloseTab:
			CloseTab()
			return nil
		case config.CmdQuit:
			Stop()
			return nil
		case config.CmdPrevTab:
			// Wrap around, allow for modulo with negative numbers
			n := NumTabs()
			SwitchTab((((curTab - 1) % n) + n) % n)
			return nil
		case config.CmdNextTab:
			SwitchTab((curTab + 1) % NumTabs())
			return nil
		case config.CmdHelp:
			Help()
			return nil
		}

		if cmd >= config.CmdTab1 && cmd <= config.CmdTab0 {
			if cmd == config.CmdTab0 {
				// Zero key goes to the last tab
				SwitchTab(NumTabs() - 1)
			} else {
				SwitchTab(int(cmd - config.CmdTab1))
			}
			return nil
		}

		// Let another element handle the event, it's not a special global key
		return event
	})
}

// Stop stops the app gracefully.
// In the future it will handle things like ongoing downloads, etc
func Stop() {
	App.Stop()
}

// NewTab opens a new tab and switches to it, displaying the
// the default empty content because there's no URL.
func NewTab() {
	// Create TextView and change curTab
	// Set the TextView options, and the changed func to App.Draw()
	// SetDoneFunc to do link highlighting
	// Add view to pages and switch to it

	// Process current tab before making a new one
	if curTab > -1 {
		// Turn off link selecting mode in the current tab
		tabs[curTab].view.Highlight("")
		// Save bottomBar state
		tabs[curTab].saveBottomBar()
	}

	curTab = NumTabs()

	tabs = append(tabs, makeNewTab())
	temp := newTabPage // Copy
	setPage(tabs[curTab], &temp)
	tabs[curTab].addToHistory("about:newtab")
	tabs[curTab].history.pos = 0 // Manually set as first page

	browser.AddTab(
		strconv.Itoa(curTab),
		makeTabLabel(strconv.Itoa(curTab+1)),
		makeContentLayout(tabs[curTab].view, leftMargin()),
	)
	browser.SetCurrentTab(strconv.Itoa(curTab))
	App.SetFocus(tabs[curTab].view)

	bottomBar.SetLabel("")
	bottomBar.SetText("")
	tabs[curTab].saveBottomBar()

	// Draw just in case
	App.Draw()
}

// CloseTab closes the current tab and switches to the one to its left.
func CloseTab() {
	// Basically the NewTab() func inverted

	// TODO: Support closing middle tabs, by renumbering all the maps
	// So that tabs to the right of the closed tabs point to the right places
	// For now you can only close the right-most tab
	if curTab != NumTabs()-1 {
		return
	}

	if NumTabs() <= 1 {
		// There's only one tab open, close the app instead
		Stop()
		return
	}

	tabs = tabs[:len(tabs)-1]
	browser.RemoveTab(strconv.Itoa(curTab))

	if curTab <= 0 {
		curTab = NumTabs() - 1
	} else {
		curTab--
	}

	browser.SetCurrentTab(strconv.Itoa(curTab)) // Go to previous page
	// Restore previous tab's state
	tabs[curTab].applyAll()

	App.SetFocus(tabs[curTab].view)

	// Just in case
	App.Draw()
}

// SwitchTab switches to a specific tab, using its number, 0-indexed.
// The tab numbers are clamped to the end, so for example numbers like -5 and 1000 are still valid.
// This means that calling something like SwitchTab(curTab - 1) will never cause an error.
func SwitchTab(tab int) {
	if tab < 0 {
		tab = 0
	}
	if tab > NumTabs()-1 {
		tab = NumTabs() - 1
	}

	// Save current tab attributes
	if curTab > -1 {
		// Save bottomBar state
		tabs[curTab].saveBottomBar()
	}

	curTab = tab % NumTabs()

	// Display tab
	reformatPageAndSetView(tabs[curTab], tabs[curTab].page)
	browser.SetCurrentTab(strconv.Itoa(curTab))
	tabs[curTab].applyAll()

	App.SetFocus(tabs[curTab].view)

	// Just in case
	App.Draw()
}

func Reload() {
	if tabs[curTab].page.URL == "about:newtab" && config.CustomNewTab {
		// Re-render new tab, similar to Init()
		newTabContent := getNewTabContent()
		tmpTermW := termW
		renderedNewTabContent, newTabLinks := renderer.RenderGemini(newTabContent, textWidth(), false)
		newTabPage = structs.Page{
			Raw:       newTabContent,
			Content:   renderedNewTabContent,
			Links:     newTabLinks,
			URL:       "about:newtab",
			TermWidth: tmpTermW,
			Mediatype: structs.TextGemini,
		}
		temp := newTabPage // Copy
		setPage(tabs[curTab], &temp)
		return
	}

	if !tabs[curTab].hasContent() {
		return
	}

	go func(t *tab) {
		cache.RemovePage(tabs[curTab].page.URL)
		handleURL(t, t.page.URL, 0) // goURL is not used bc history shouldn't be added to
		if t == tabs[curTab] {
			// Display the bottomBar state that handleURL set
			t.applyBottomBar()
		}
	}(tabs[curTab])
}

// URL loads and handles the provided URL for the current tab.
// It should be an absolute URL.
func URL(u string) {
	t := tabs[curTab]
	if strings.HasPrefix(u, "about:") {
		if final, ok := handleAbout(t, u); ok {
			t.addToHistory(final)
		}
		return
	}

	go goURL(t, fixUserURL(u))
}

func RenderFromString(str string) {
	t := tabs[curTab]
	page, _ := renderPageFromString(str)
	setPage(t, page)
}

func renderPageFromString(str string) (*structs.Page, bool) {
	rendered, links := renderer.RenderGemini(str, textWidth(), false)
	page := &structs.Page{
		Mediatype: structs.TextGemini,
		Raw:       str,
		Content:   rendered,
		Links:     links,
		TermWidth: termW,
	}

	return page, true
}

func NumTabs() int {
	return len(tabs)
}
