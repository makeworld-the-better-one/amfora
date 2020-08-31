//nolint
package display

import (
	"io/ioutil"
	"path/filepath"

	"github.com/makeworld-the-better-one/amfora/config"
)

var defaultNewTabContent = `# New Tab

You've opened a new tab. Use the bar at the bottom to browse around. You can start typing in it by pressing the space key.

Press the ? key at any time to bring up the help, and see other keybindings. Most are what you expect.

You can customize this page by creating the Gemtext file

` + "```" + `
$XDG_CONFIG_HOME/amfora/new_tab.gmi
` + "```" + `

Happy browsing!

=> about:bookmarks Bookmarks

=> //gemini.circumlunar.space Project Gemini
=> https://github.com/makeworld-the-better-one/amfora Amfora homepage [HTTPS]
`

// Read the new tab content from a file if it exists or fallback to a default page.
func getNewTabContent() string {
	newTabFile := filepath.Join(config.ConfigDir, "new_tab.gmi")
	data, err := ioutil.ReadFile(newTabFile)
	if err == nil {
		return string(data)
	} else {
		return defaultNewTabContent
	}
}
