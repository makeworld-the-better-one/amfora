// Package config initializes all files required for Amfora, even those used by
// other packages. It also reads in the config file and initializes a Viper and
// the theme
//nolint:golint,goerr113
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/gdamore/tcell"
	"github.com/makeworld-the-better-one/amfora/cache"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/rkoesters/xdg/basedir"
	"github.com/rkoesters/xdg/userdirs"
	"github.com/spf13/viper"
	"gitlab.com/tslocum/cview"
)

var amforaAppData string // Where amfora files are stored on Windows - cached here
var configDir string
var configPath string

var NewTabPath string
var CustomNewTab bool

var TofuStore = viper.New()
var tofuDBDir string
var tofuDBPath string

// Bookmarks

var BkmkStore = viper.New()
var bkmkDir string
var bkmkPath string

var DownloadsDir string

// Subscriptions
var subscriptionDir string
var SubscriptionPath string

// Command for opening HTTP(S) URLs in the browser, from "a-general.http" in config.
var HTTPCommand []string

func Init() error {

	// *** Set paths ***

	home, err := homedir.Dir()
	if err != nil {
		return err
	}
	// Store AppData path
	if runtime.GOOS == "windows" { //nolint:goconst
		appdata, ok := os.LookupEnv("APPDATA")
		if ok {
			amforaAppData = filepath.Join(appdata, "amfora")
		} else {
			amforaAppData = filepath.Join(home, filepath.FromSlash("AppData/Roaming/amfora/"))
		}
	}

	// Store config directory and file paths
	if runtime.GOOS == "windows" {
		configDir = amforaAppData
	} else {
		// Unix / POSIX system
		configDir = filepath.Join(basedir.ConfigHome, "amfora")
	}
	configPath = filepath.Join(configDir, "config.toml")

	// Search for a custom new tab
	NewTabPath = filepath.Join(configDir, "newtab.gmi")
	CustomNewTab = false
	if _, err := os.Stat(NewTabPath); err == nil {
		CustomNewTab = true
	}

	// Store TOFU db directory and file paths
	if runtime.GOOS == "windows" {
		// Windows just stores it in APPDATA along with other stuff
		tofuDBDir = amforaAppData
	} else {
		// XDG cache dir on POSIX systems
		tofuDBDir = filepath.Join(basedir.CacheHome, "amfora")
	}
	tofuDBPath = filepath.Join(tofuDBDir, "tofu.toml")

	// Store bookmarks dir and path
	if runtime.GOOS == "windows" {
		// Windows just keeps it in APPDATA along with other Amfora files
		bkmkDir = amforaAppData
	} else {
		// XDG data dir on POSIX systems
		bkmkDir = filepath.Join(basedir.DataHome, "amfora")
	}
	bkmkPath = filepath.Join(bkmkDir, "bookmarks.toml")

	// Feeds dir and path
	if runtime.GOOS == "windows" {
		// In APPDATA beside other Amfora files
		subscriptionDir = amforaAppData
	} else {
		// XDG data dir on POSIX systems
		xdg_data, ok := os.LookupEnv("XDG_DATA_HOME")
		if ok && strings.TrimSpace(xdg_data) != "" {
			subscriptionDir = filepath.Join(xdg_data, "amfora")
		} else {
			// Default to ~/.local/share/amfora
			subscriptionDir = filepath.Join(home, ".local", "share", "amfora")
		}
	}
	SubscriptionPath = filepath.Join(subscriptionDir, "subscriptions.json")

	// *** Create necessary files and folders ***

	// Config
	err = os.MkdirAll(configDir, 0755)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(configPath, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0666)
	if err == nil {
		// Config file doesn't exist yet, write the default one
		_, err = f.Write(defaultConf)
		if err != nil {
			f.Close()
			return err
		}
		f.Close()
	}
	// TOFU
	err = os.MkdirAll(tofuDBDir, 0755)
	if err != nil {
		return err
	}
	f, err = os.OpenFile(tofuDBPath, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0666)
	if err == nil {
		f.Close()
	}
	// Bookmarks
	err = os.MkdirAll(bkmkDir, 0755)
	if err != nil {
		return err
	}
	f, err = os.OpenFile(bkmkPath, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0666)
	if err == nil {
		f.Close()
	}
	// Feeds
	err = os.MkdirAll(subscriptionDir, 0755)
	if err != nil {
		return err
	}

	// *** Downloads paths, setup, and creation ***

	// Setup downloads dir
	if viper.GetString("a-general.downloads") == "" {
		// Find default Downloads dir
		// This seems to work for all OSes?
		if userdirs.Download == "" {
			DownloadsDir = filepath.Join(home, "Downloads")
		} else {
			DownloadsDir = userdirs.Download
		}
		// Create it just in case
		err = os.MkdirAll(DownloadsDir, 0755)
		if err != nil {
			return fmt.Errorf("downloads path could not be created: %s", DownloadsDir)
		}
	} else {
		// Validate path
		dDir := viper.GetString("a-general.downloads")
		di, err := os.Stat(dDir)
		if err == nil {
			if !di.IsDir() {
				return fmt.Errorf("downloads path specified is not a directory: %s", dDir)
			}
		} else if os.IsNotExist(err) {
			// Try to create path
			err = os.MkdirAll(dDir, 0755)
			if err != nil {
				return fmt.Errorf("downloads path could not be created: %s", dDir)
			}
		} else {
			// Some other error
			return fmt.Errorf("couldn't access downloads directory: %s", dDir)
		}
		DownloadsDir = dDir
	}

	// *** Setup vipers ***

	TofuStore.SetConfigFile(tofuDBPath)
	TofuStore.SetConfigType("toml")
	err = TofuStore.ReadInConfig()
	if err != nil {
		return err
	}

	BkmkStore.SetConfigFile(bkmkPath)
	BkmkStore.SetConfigType("toml")
	err = BkmkStore.ReadInConfig()
	if err != nil {
		return err
	}
	BkmkStore.Set("DO NOT TOUCH", true)
	err = BkmkStore.WriteConfig()
	if err != nil {
		return err
	}

	// Setup main config

	viper.SetDefault("a-general.home", "gemini.circumlunar.space")
	viper.SetDefault("a-general.auto_redirect", false)
	viper.SetDefault("a-general.http", "default")
	viper.SetDefault("a-general.search", "gus.guru/search")
	viper.SetDefault("a-general.color", true)
	viper.SetDefault("a-general.ansi", true)
	viper.SetDefault("a-general.bullets", true)
	viper.SetDefault("a-general.show_link", false)
	viper.SetDefault("a-general.left_margin", 0.15)
	viper.SetDefault("a-general.max_width", 100)
	viper.SetDefault("a-general.downloads", "")
	viper.SetDefault("a-general.page_max_size", 2097152)
	viper.SetDefault("a-general.page_max_time", 10)
	viper.SetDefault("a-general.emoji_favicons", false)
	viper.SetDefault("keybindings.bind_reload", []string{"R", "Ctrl-R"})
	viper.SetDefault("keybindings.bind_home", "Backspace")
	viper.SetDefault("keybindings.bind_bookmarks", "Ctrl-B")
	viper.SetDefault("keybindings.bind_add_bookmark", "Ctrl-D")
	viper.SetDefault("keybindings.bind_sub", "Ctrl-A")
	viper.SetDefault("keybindings.bind_add_sub", "Ctrl-X")
	viper.SetDefault("keybindings.bind_save", "Ctrl-S")
	viper.SetDefault("keybindings.bind_pgup", []string{"PgUp", "u"})
	viper.SetDefault("keybindings.bind_pgdn", []string{"PgDn", "d"})
	viper.SetDefault("keybindings.bind_bottom", "Space")
	viper.SetDefault("keybindings.bind_edit", "e")
	viper.SetDefault("keybindings.bind_back", []string{"b", "Alt-Left"})
	viper.SetDefault("keybindings.bind_forward", []string{"f", "Alt-Right"})
	viper.SetDefault("keybindings.bind_new_tab", "Ctrl-T")
	viper.SetDefault("keybindings.bind_close_tab", "Ctrl-W")
	viper.SetDefault("keybindings.bind_next_tab", "F2")
	viper.SetDefault("keybindings.bind_prev_tab", "F1")
	viper.SetDefault("keybindings.bind_quit", []string{"Ctrl-C", "Ctrl-Q", "q"})
	viper.SetDefault("keybindings.bind_help", "?")
	viper.SetDefault("keybindings.bind_link1", "1")
	viper.SetDefault("keybindings.bind_link2", "2")
	viper.SetDefault("keybindings.bind_link3", "3")
	viper.SetDefault("keybindings.bind_link4", "4")
	viper.SetDefault("keybindings.bind_link5", "5")
	viper.SetDefault("keybindings.bind_link6", "6")
	viper.SetDefault("keybindings.bind_link7", "7")
	viper.SetDefault("keybindings.bind_link8", "8")
	viper.SetDefault("keybindings.bind_link9", "9")
	viper.SetDefault("keybindings.bind_link0", "0")
	viper.SetDefault("keybindings.bind_tab1", "!")
	viper.SetDefault("keybindings.bind_tab2", "@")
	viper.SetDefault("keybindings.bind_tab3", "#")
	viper.SetDefault("keybindings.bind_tab4", "$")
	viper.SetDefault("keybindings.bind_tab5", "%")
	viper.SetDefault("keybindings.bind_tab6", "^")
	viper.SetDefault("keybindings.bind_tab7", "&")
	viper.SetDefault("keybindings.bind_tab8", "*")
	viper.SetDefault("keybindings.bind_tab9", "(")
	viper.SetDefault("keybindings.bind_tab0", ")")
	viper.SetDefault("keybindings.shift_numbers", "")
	viper.SetDefault("url-handlers.other", "off")
	viper.SetDefault("cache.max_size", 0)
	viper.SetDefault("cache.max_pages", 20)
	viper.SetDefault("subscriptions.popup", true)
	viper.SetDefault("subscriptions.update_interval", 1800)
	viper.SetDefault("subscriptions.workers", 3)
	viper.SetDefault("subscriptions.entries_per_page", 20)

	viper.SetConfigFile(configPath)
	viper.SetConfigType("toml")
	err = viper.ReadInConfig()
	if err != nil {
		return err
	}

	// Setup the key bindings:
	KeyInit()

	// Setup cache from config
	cache.SetMaxSize(viper.GetInt("cache.max_size"))
	cache.SetMaxPages(viper.GetInt("cache.max_pages"))

	// Setup theme
	configTheme := viper.Sub("theme")
	if configTheme != nil {
		for k, v := range configTheme.AllSettings() {
			colorStr, ok := v.(string)
			if !ok {
				return fmt.Errorf(`value for "%s" is not a string: %v`, k, v)
			}
			color := tcell.GetColor(strings.ToLower(colorStr))
			if color == tcell.ColorDefault {
				return fmt.Errorf(`invalid color format for "%s": %s`, k, colorStr)
			}
			SetColor(k, color)
		}
	}
	if viper.GetBool("a-general.color") {
		cview.Styles.PrimitiveBackgroundColor = GetColor("bg")
	} // Otherwise it's black by default

	// Parse HTTP command
	HTTPCommand = viper.GetStringSlice("a-general.http")
	if len(HTTPCommand) == 0 {
		// Not a string array, interpret as a string instead
		// Split on spaces to maintain compatibility with old versions
		// The new better way to is to just define a string array in config
		HTTPCommand = strings.Fields(viper.GetString("a-general.http"))
	}

	return nil
}
