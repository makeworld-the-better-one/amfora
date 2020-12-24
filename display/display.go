package display

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/gdamore/tcell"
	"github.com/makeworld-the-better-one/amfora/cache"
	"github.com/makeworld-the-better-one/amfora/config"
	"github.com/makeworld-the-better-one/amfora/renderer"
	"github.com/makeworld-the-better-one/amfora/structs"
	"github.com/makeworld-the-better-one/go-gemini"
	"github.com/spf13/viper"
	"gitlab.com/tslocum/cview"
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
var tabPages = cview.NewPages()

// The tabs at the top with titles
var tabRow = cview.NewTextView().
	SetDynamicColors(true).
	SetRegions(true).
	SetScrollable(true).
	SetWrap(false).
	SetHighlightedFunc(func(added, removed, remaining []string) {
		// There will always only be one string in added - never multiple highlights
		// Remaining should always be empty
		i, _ := strconv.Atoi(added[0])
		tabPages.SwitchToPage(strconv.Itoa(i)) // Tab names are just numbers, zero-indexed
	})

// Root layout
var layout = cview.NewFlex().
	SetDirection(cview.FlexRow)

var newTabPage structs.Page
var versionPage structs.Page

var App = cview.NewApplication().
	EnableMouse(false).
	SetRoot(layout, true).
	SetAfterResizeFunc(func(width int, height int) {
		// Store for calculations
		termW = width
		termH = height

		// Make sure the current tab content is reformatted when the terminal size changes
		go func(t *tab) {
			t.reformatMu.Lock() // Only one reformat job per tab
			defer t.reformatMu.Unlock()
			// Use the current tab, but don't affect other tabs if the user switches tabs
			reformatPageAndSetView(t, t.page)
		}(tabs[curTab])
	})

func Init(version, commit, builtBy string) {
	versionContent := fmt.Sprintf(
		"# Amfora Version Info\n\nAmfora:   %s\nCommit:   %s\nBuilt by: %s",
		version, commit, builtBy,
	)
	renderVersionContent, versionLinks := renderer.RenderGemini(versionContent, textWidth(), leftMargin(), false)
	versionPage = structs.Page{
		Raw:       versionContent,
		Content:   renderVersionContent,
		Links:     versionLinks,
		URL:       "about:version",
		Width:     -1, // Force reformatting on first display
		Mediatype: structs.TextGemini,
	}

	tabRow.SetChangedFunc(func() {
		App.Draw()
	})

	helpInit()

	layout.
		AddItem(tabRow, 1, 1, false).
		AddItem(nil, 1, 1, false). // One line of empty space above the page
		AddItem(tabPages, 0, 1, true).
		AddItem(nil, 1, 1, false). // One line of empty space before bottomBar
		AddItem(bottomBar, 1, 1, false)

	if viper.GetBool("a-general.color") {
		layout.SetBackgroundColor(config.GetColor("bg"))
		tabRow.SetBackgroundColor(config.GetColor("bg"))

		bottomBar.SetBackgroundColor(config.GetColor("bottombar_bg"))
		bottomBar.
			SetLabelColor(config.GetColor("bottombar_label")).
			SetFieldBackgroundColor(config.GetColor("bottombar_bg")).
			SetFieldTextColor(config.GetColor("bottombar_text"))
	} else {
		bottomBar.SetBackgroundColor(tcell.ColorWhite)
		bottomBar.
			SetLabelColor(tcell.ColorBlack).
			SetFieldBackgroundColor(tcell.ColorWhite).
			SetFieldTextColor(tcell.ColorBlack)
	}
	bottomBar.SetDoneFunc(func(key tcell.Key) {
		tab := curTab

		tabs[tab].saveScroll()

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
			if query[0] == '.' && tabs[tab].hasContent() {
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
					if (strings.Contains(query, " ") && !hasSpaceisURL.MatchString(query)) ||
						(!strings.HasPrefix(query, "//") && !strings.Contains(query, "://") &&
							!strings.Contains(query, ".")) {
						// Has a space and follows regex, OR
						// doesn't start with "//", contain "://", and doesn't have a dot either.
						// Then it's a search

						u := viper.GetString("a-general.search") + "?" + gemini.QueryEscape(query)
						cache.RemovePage(u) // Don't use the cached version of the search
						URL(u)
					} else {
						// Full URL
						cache.RemovePage(query) // Don't use cached version for manually entered URL
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
	renderedNewTabContent, newTabLinks := renderer.RenderGemini(newTabContent, textWidth(), leftMargin(), false)
	newTabPage = structs.Page{
		Raw:       newTabContent,
		Content:   renderedNewTabContent,
		Links:     newTabLinks,
		URL:       "about:newtab",
		Width:     -1, // Force reformatting on first display
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

		if tabs[curTab].mode == tabModeDone {
			// All the keys and operations that can only work while NOT loading

			// History arrow keys
			if event.Modifiers() == tcell.ModAlt {
				if event.Key() == tcell.KeyLeft {
					histBack(tabs[curTab])
					return nil
				}
				if event.Key() == tcell.KeyRight {
					histForward(tabs[curTab])
					return nil
				}
			}

			//nolint:exhaustive
			switch event.Key() {
			case tcell.KeyCtrlR:
				Reload()
				return nil
			case tcell.KeyCtrlH:
				URL(viper.GetString("a-general.home"))
				return nil
			case tcell.KeyCtrlB:
				Bookmarks(tabs[curTab])
				tabs[curTab].addToHistory("about:bookmarks")
				return nil
			case tcell.KeyCtrlD:
				go addBookmark()
				return nil
			case tcell.KeyPgUp:
				tabs[curTab].pageUp()
				return nil
			case tcell.KeyPgDn:
				tabs[curTab].pageDown()
				return nil
			case tcell.KeyCtrlS:
				if tabs[curTab].hasContent() {
					savePath, err := downloadPage(tabs[curTab].page)
					if err != nil {
						Error("Download Error", fmt.Sprintf("Error saving page content: %v", err))
					} else {
						Info(fmt.Sprintf("Page content saved to %s. ", savePath))
					}
				} else {
					Info("The current page has no content, so it couldn't be downloaded.")
				}
				return nil
			case tcell.KeyCtrlA:
				Subscriptions(tabs[curTab], "about:subscriptions")
				tabs[curTab].addToHistory("about:subscriptions")
				return nil
			case tcell.KeyCtrlX:
				go addSubscription()
				return nil
			case tcell.KeyRune:
				// Regular key was sent
				switch string(event.Rune()) {
				case " ":
					// Space starts typing, like Bombadillo
					bottomBar.SetLabel("[::b]URL/Num./Search: [::-]")
					bottomBar.SetText("")
					// Don't save bottom bar, so that whenever you switch tabs, it's not in that mode
					App.SetFocus(bottomBar)
					return nil
				case "e":
					// Letter e allows to edit current URL
					bottomBar.SetLabel("[::b]Edit URL: [::-]")
					bottomBar.SetText(tabs[curTab].page.URL)
					App.SetFocus(bottomBar)
					return nil
				case "R":
					Reload()
					return nil
				case "b":
					histBack(tabs[curTab])
					return nil
				case "f":
					histForward(tabs[curTab])
					return nil
				case "u":
					tabs[curTab].pageUp()
					return nil
				case "d":
					tabs[curTab].pageDown()
					return nil
				}

				// Number key: 1-9, 0
				i, err := strconv.Atoi(string(event.Rune()))
				if err == nil {
					if i == 0 {
						i = 10 // 0 key is for link 10
					}
					if i <= len(tabs[curTab].page.Links) && i > 0 {
						// It's a valid link number
						followLink(tabs[curTab], tabs[curTab].page.URL, tabs[curTab].page.Links[i-1])
						return nil
					}
				}
			}
		}

		// All the keys and operations that can work while a tab IS loading

		//nolint:exhaustive
		switch event.Key() {
		case tcell.KeyCtrlT:
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
		case tcell.KeyCtrlW:
			CloseTab()
			return nil
		case tcell.KeyCtrlQ:
			Stop()
			return nil
		case tcell.KeyCtrlC:
			Stop()
			return nil
		case tcell.KeyF1:
			// Wrap around, allow for modulo with negative numbers
			n := NumTabs()
			SwitchTab((((curTab - 1) % n) + n) % n)
			return nil
		case tcell.KeyF2:
			SwitchTab((curTab + 1) % NumTabs())
			return nil
		case tcell.KeyRune:
			// Regular key was sent

			if num, err := config.KeyToNum(event.Rune()); err == nil {
				// It's a Shift+Num key
				if num == 0 {
					// Zero key goes to the last tab
					SwitchTab(NumTabs() - 1)
				} else {
					SwitchTab(num - 1)
				}
				return nil
			}

			switch string(event.Rune()) {
			case "q":
				Stop()
				return nil
			case "?":
				Help()
				return nil
			}
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
		tabs[curTab].saveScroll()
	}

	curTab = NumTabs()

	tabs = append(tabs, makeNewTab())
	temp := newTabPage // Copy
	setPage(tabs[curTab], &temp)
	tabs[curTab].addToHistory("about:newtab")
	tabs[curTab].history.pos = 0 // Manually set as first page

	tabPages.AddAndSwitchToPage(strconv.Itoa(curTab), tabs[curTab].view, true)
	App.SetFocus(tabs[curTab].view)

	// Add tab number to the actual place where tabs are show on the screen
	// Tab regions are 0-indexed but text displayed on the screen starts at 1
	if viper.GetBool("a-general.color") {
		fmt.Fprintf(tabRow, `["%d"][%s]  %d  [%s][""]|`,
			curTab,
			config.GetColorString("tab_num"),
			curTab+1,
			config.GetColorString("tab_divider"),
		)
	} else {
		fmt.Fprintf(tabRow, `["%d"]  %d  [""]|`, curTab, curTab+1)
	}
	tabRow.Highlight(strconv.Itoa(curTab)).ScrollToHighlight()

	bottomBar.SetLabel("")
	bottomBar.SetText("")
	tabs[curTab].saveBottomBar()

	// Draw just in case
	App.Draw()
}

// CloseTab closes the current tab and switches to the one to its left.
func CloseTab() {
	// Basically the NewTab() func inverted

	if NumTabs() <= 1 {
		// There's only one tab open, close the app instead
		Stop()
		return
	}

	// Remove the page of the closed tab and the pages of all subsequent tabs
	for i := curTab; i < NumTabs(); i++ {
		tabPages.RemovePage(strconv.Itoa(i))
	}
	// Remove the closed tab
	tabs = append(tabs[:curTab], tabs[curTab+1:]...)

	// Re-add the pages of the subsequent tabs
	for i := curTab; i < NumTabs(); i++ {
		tabPages.AddPage(strconv.Itoa(i), tabs[i].view, true, false)
	}

	// Switch to the tab on the left
	curTab--
	if curTab < 0 {
		curTab = 0
	}

	tabPages.SwitchToPage(strconv.Itoa(curTab)) // Go to previous page
	rewriteTabRow()
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
		tabs[curTab].saveScroll()
	}

	curTab = tab % NumTabs()

	// Display tab
	reformatPageAndSetView(tabs[curTab], tabs[curTab].page)
	tabPages.SwitchToPage(strconv.Itoa(curTab))
	tabRow.Highlight(strconv.Itoa(curTab)).ScrollToHighlight()
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
		renderedNewTabContent, newTabLinks := renderer.RenderGemini(newTabContent, textWidth(), leftMargin(), false)
		newTabPage = structs.Page{
			Raw:       newTabContent,
			Content:   renderedNewTabContent,
			Links:     newTabLinks,
			URL:       "about:newtab",
			Width:     tmpTermW,
			Mediatype: structs.TextGemini,
		}
		temp := newTabPage // Copy
		setPage(tabs[curTab], &temp)
		return
	}

	if !tabs[curTab].hasContent() {
		return
	}

	parsed, _ := url.Parse(tabs[curTab].page.URL)
	go func(t *tab) {
		cache.RemovePage(tabs[curTab].page.URL)
		cache.RemoveFavicon(parsed.Host)
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

	if !strings.HasPrefix(u, "//") && !strings.HasPrefix(u, "gemini://") && !strings.Contains(u, "://") {
		// Assume it's a Gemini URL
		u = "gemini://" + u
	}
	go goURL(t, u)
}

func NumTabs() int {
	return len(tabs)
}
