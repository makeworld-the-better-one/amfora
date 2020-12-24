package display

import (
	"strconv"
	"strings"

	"github.com/gdamore/tcell"
	"gitlab.com/tslocum/cview"
)

var helpCells = strings.TrimSpace(`
?|Bring up this help. You can scroll!
Esc|Leave the help
Arrow keys, h/j/k/l|Scroll and move a page.
PgUp, u|Go up a page in document
PgDn, d|Go down a page in document
g|Go to top of document
G|Go to bottom of document
Tab|Navigate to the next item in a popup.
Shift-Tab|Navigate to the previous item in a popup.
b, Alt-Left|Go back in the history
f, Alt-Right|Go forward in the history
spacebar|Open bar at the bottom - type a URL, link number, search term.
|You can also type two dots (..) to go up a directory in the URL.
|Typing new:N will open link number N in a new tab
|instead of the current one.
Numbers|Go to links 1-10 respectively.
e|Edit current URL
Enter, Tab|On a page this will start link highlighting.
|Press Tab and Shift-Tab to pick different links.
|Press Enter again to go to one, or Esc to stop.
Shift-NUMBER|Go to a specific tab.
Shift-0, )|Go to the last tab.
F1|Previous tab
F2|Next tab
Ctrl-H|Go home
Ctrl-T|New tab, or if a link is selected,
|this will open the link in a new tab.
Ctrl-W|Close tab.
Ctrl-R, R|Reload a page, discarding the cached version.
|This can also be used if you resize your terminal.
Ctrl-B|View bookmarks
Ctrl-D|Add, change, or remove a bookmark for the current page.
Ctrl-S|Save the current page to your downloads.
Ctrl-A|View subscriptions
Ctrl-X|Add or update a subscription
q, Ctrl-Q|Quit
Ctrl-C|Hard quit. This can be used when in the middle of downloading,
|for example.
`)

var helpTable = cview.NewTable().
	SetSelectable(false, false).
	SetBorders(false).
	SetScrollBarVisibility(cview.ScrollBarNever)

// Help displays the help and keybindings.
func Help() {
	helpTable.ScrollToBeginning()
	tabPages.SwitchToPage("help")
	App.SetFocus(helpTable)
	App.Draw()
}

func helpInit() {
	// Populate help table
	helpTable.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEsc || key == tcell.KeyEnter {
			tabPages.SwitchToPage(strconv.Itoa(curTab))
			App.SetFocus(tabs[curTab].view)
			App.Draw()
		}
	})
	rows := strings.Count(helpCells, "\n") + 1
	cells := strings.Split(
		strings.ReplaceAll(helpCells, "\n", "|"),
		"|")
	cell := 0
	extraRows := 0 // Rows continued from the previous, without spacing
	for r := 0; r < rows; r++ {
		for c := 0; c < 2; c++ {
			var tableCell *cview.TableCell
			if c == 0 {
				// First column, the keybinding
				tableCell = cview.NewTableCell(" " + cells[cell]).
					SetAttributes(tcell.AttrBold).
					SetAlign(cview.AlignLeft)
			} else {
				tableCell = cview.NewTableCell(" " + cells[cell])
			}
			if c == 0 && cells[cell] == "" || (cell > 0 && cells[cell-1] == "" && c == 1) {
				// The keybinding column for this row was blank, meaning the explanation
				// column is continued from the previous row.
				// The row should be added without any spacing rows
				helpTable.SetCell(((2*r)-extraRows/2)-1, c, tableCell)
				extraRows++
			} else {
				helpTable.SetCell((2*r)-extraRows/2, c, tableCell) // Every other row, for readability
			}
			cell++
		}
	}
	tabPages.AddPage("help", helpTable, true, false)
}
