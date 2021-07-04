package config

//go:generate ./default.sh
var defaultConf = []byte(`# This is the default config file.
# It also shows all the default values, if you don't create the file.

# All URL values may omit the scheme and/or port, as well as the beginning double slash
# Valid URL examples:
# gemini://example.com
# //example.com
# example.com
# example.com:123


[a-general]
# Press Ctrl-H to access it
home = "gemini://gemini.circumlunar.space"

# Follow up to 5 Gemini redirects without prompting.
# A prompt is always shown after the 5th redirect and for redirects to protocols other than Gemini.
# If set to false, a prompt will be shown before following redirects.
auto_redirect = false

# What command to run to open a HTTP(S) URL.
# Set to "default" to try to guess the browser, or set to "off" to not open HTTP(S) URLs.
# If a command is set, than the URL will be added (in quotes) to the end of the command.
# A space will be prepended to the URL.
#
# The best to define a command is using a string array.
# Examples:
# http = ['firefox']
# http = ['custom-browser', '--flag', '--option=2']
# http = ['/path/with spaces/in it/firefox']
#
# Note the use of single quotes, so that backslashes will not be escaped.
# Using just a string will also work, but it is deprecated, and will degrade if
# you use paths with spaces.

http = 'default'

# Any URL that will accept a query string can be put here
search = "gemini://geminispace.info/search"

# Whether colors will be used in the terminal
color = true

# Whether ANSI color codes from the page content should be rendered
ansi = true

# Whether to replace list asterisks with unicode bullets
bullets = true

# Whether to show link after link text
show_link = false

# A number from 0 to 1, indicating what percentage of the terminal width the left margin should take up.
left_margin = 0.15

# The max number of columns to wrap a page's text to. Preformatted blocks are not wrapped.
max_width = 100

# 'downloads' is the path to a downloads folder.
# An empty value means the code will find the default downloads folder for your system.
# If the path does not exist it will be created.
# Note the use of single quotes, so that backslashes will not be escaped.
downloads = ''

# Max size for displayable content in bytes - after that size a download window pops up
page_max_size = 2097152  # 2 MiB
# Max time it takes to load a page in seconds - after that a download window pops up
page_max_time = 10

# When a scrollbar appears. "never", "auto", and "always" are the only valid values.
# "auto" means the scrollbar only appears when the page is longer than the window.
scrollbar = "auto"


[auth]
# Authentication settings
# Note the use of single quotes for values, so that backslashes will not be escaped.

[auth.certs]
# Client certificates
# Set domain name equal to path to client cert
# "example.com" = 'mycert.crt'

[auth.keys]
# Client certificate keys
# Set domain name equal to path to key for the client cert above
# "example.com" = 'mycert.key'


[keybindings]
# If you have a non-US keyboard, use bind_tab1 through bind_tab0 to
# setup the shift-number bindings: Eg, for US keyboards (the default):
# bind_tab1 = "!"
# bind_tab2 = "@"
# bind_tab3 = "#"
# bind_tab4 = "$"
# bind_tab5 = "%"
# bind_tab6 = "^"
# bind_tab7 = "&"
# bind_tab8 = "*"
# bind_tab9 = "("
# bind_tab0 = ")"

# Whitespace is not allowed in any of the keybindings! Use 'Space' and 'Tab' to bind to those keys.
# Multiple keys can be bound to one command, just use a TOML array.
# To add the Alt modifier, the binding must start with Alt-, should be reasonably universal
# Ctrl- won't work on all keys, see this for a list:
# https://github.com/gdamore/tcell/blob/cb1e5d6fa606/key.go#L83

# An example of a TOML array for multiple keys being bound to one command is the default
# binding for reload:
# bind_reload = ["R","Ctrl-R"]
# One thing to note here is that "R" is capitalization sensitive, so it means shift-r.
# "Ctrl-R" means both ctrl-r and ctrl-shift-R (this is a quirk of what ctrl-r means on
# an ANSI terminal)

# The default binding for opening the bottom bar for entering a URL or link number is:
# bind_bottom = "Space"
# This is how to get the Spacebar as a keybinding, if you try to use " ", it won't work.
# And, finally, an example of a simple, unmodified character is:
# bind_edit = "e"
# This binds the "e" key to the command to edit the current URL.

# The bind_link[1-90] options are for the commands to go to the first 10 links on a page,
# typically these are bound to the number keys:
# bind_link1 = "1"
# bind_link2 = "2"
# bind_link3 = "3"
# bind_link4 = "4"
# bind_link5 = "5"
# bind_link6 = "6"
# bind_link7 = "7"
# bind_link8 = "8"
# bind_link9 = "9"
# bind_link0 = "0"

# All keybindings:
#
# bind_bottom
# bind_edit
# bind_home
# bind_bookmarks
# bind_add_bookmark
# bind_save
# bind_reload
# bind_back
# bind_forward
# bind_moveup
# bind_movedown
# bind_moveleft
# bind_moveright
# bind_pgup
# bind_pgdn
# bind_new_tab
# bind_close_tab
# bind_next_tab
# bind_prev_tab
# bind_quit
# bind_help
# bind_sub: for viewing the subscriptions page
# bind_add_sub
# bind_copy_page_url
# bind_copy_target_url
# bind_beginning: moving to beginning of page (top left)
# bind_end: same but the for the end (bottom left)

[url-handlers]
# Allows setting the commands to run for various URL schemes.
# To pass the URL to the program specify {{url}}.
# E.g. to open FTP URLs with FileZilla set the following key:
#   ftp = 'filezilla {{url}}'
# You can set any scheme to "off" or "" to disable handling it, or
# just leave the key unset.
#
# DO NOT use this for setting the HTTP command.
# Use the http setting in the "a-general" section above.
#
# NOTE: These settings are overrided by the ones in the proxies section.
# Note the use of single quotes, so that backslashes will not be escaped.

# This is a special key that defines the handler for all URL schemes for which
# no handler is defined.
other = 'off'


# [[mediatype-handlers]] section
# ---------------------------------
#
# Specify what applications will open certain media types.
# By default your default application will be used to open the file when you select "Open".
# You only need to configure this section if you want to override your default application,
# or do special things like streaming.
#
# Note the use of single quotes for commands, so that backslashes will not be escaped.
#
#
# To open jpeg files with the feh command:
#
# [[mediatype-handlers]]
# cmd = ['feh']
# types = ["image/jpeg"]
#
# Each command that you specify must come under its own [[mediatype-handlers]]. You may
# specify as many [[mediatype-handlers]] as you want to setup multiple commands.
#
# If the subtype is omitted then the specified command will be used for the
# entire type:
#
# [[mediatype-handlers]]
# command = ['vlc', '--flag']
# types = ["audio", "video"]
#
# A catch-all handler can by specified with "*".
# Note that there are already catch-all handlers in place for all OSes,
# that open the file using your default application. This is only if you
# want to override that.
#
# [[mediatype-handlers]]
# cmd = ['some-command']
# types = [
#         "application/pdf",
#         "*",
# ]
#
# You can also choose to stream the data instead of downloading it all before
# opening it. This is especially useful for large video or audio files, as
# well as radio streams, which will never complete. You can do this like so:
#
# [[mediatype-handlers]]
# cmd = ['vlc', '-']
# types = ["audio", "video"]
# stream = true
#
# This uses vlc to stream all video and audio content.
# By default stream is set to off for all handlers
#
#
# If you want to always open a type in its viewer without the download or open
# prompt appearing, you can add no_prompt = true
#
# [[mediatype-handlers]]
# cmd = ['feh']
# types = ["image"]
# no_prompt = true
#
# Note: Multiple handlers cannot be defined for the same full media type, but
# still there needs to be an order for which handlers are used. The following
# order applies regardless of the order written in the config:
#
# 1. Full media type: "image/jpeg"
# 2. Just type: "image"
# 3. Catch-all: "*"


[cache]
# Options for page cache - which is only for text pages
# Increase the cache size to speed up browsing at the expense of memory
# Zero values mean there is no limit

max_size = 0  # Size in bytes
max_pages = 30 # The maximum number of pages the cache will store

# How long a page will stay in cache, in seconds.
timeout = 1800 # 30 mins

[proxies]
# Allows setting a Gemini proxy for different schemes.
# The settings are similar to the url-handlers section above.
# E.g. to open a gopher page by connecting to a Gemini proxy server:
#   gopher = "example.com:123"
#
# Port 1965 is assumed if no port is specified.
#
# NOTE: These settings override any external handlers specified in
# the url-handlers section.
#
# Note that HTTP and HTTPS are treated as separate protocols here.


[subscriptions]
# For tracking feeds and pages

# Whether a pop-up appears when viewing a potential feed
popup = true

# How often to check for updates to subscriptions in the background, in seconds.
# Set it to 0 to disable this feature. You can still update individual feeds
# manually, or restart the browser.
#
# Note Amfora will check for updates on browser start no matter what this setting is.
update_interval = 1800 # 30 mins

# How many subscriptions can be checked at the same time when updating.
# If you have many subscriptions you may want to increase this for faster
# update times. Any value below 1 will be corrected to 1.
workers = 3

# The number of subscription updates displayed per page.
entries_per_page = 20


[theme]
# This section is for changing the COLORS used in Amfora.
# These colors only apply if 'color' is enabled above.
# Colors can be set using a W3C color name, or a hex value such as "#ffffff".

# Note that not all colors will work on terminals that do not have truecolor support.
# If you want to stick to the standard 16 or 256 colors, you can get
# a list of those here: https://jonasjacek.github.io/colors/
# DO NOT use the names from that site, just the hex codes.

# Definitions:
#   bg = background
#   fg = foreground
#   dl = download
#   btn = button
#   hdg = heading
#   bkmk = bookmark
#   modal = a popup window/box in the middle of the screen

# EXAMPLES:
# hdg_1 = "green"
# hdg_2 = "#5f0000"

# Available keys to set:

# bg: background for pages, tab row, app in general
# tab_num: The number/highlight of the tabs at the top
# tab_divider: The color of the divider character between tab numbers: |
# bottombar_label: The color of the prompt that appears when you press space
# bottombar_text: The color of the text you type
# bottombar_bg
# scrollbar: The scrollbar that appears on the right for long pages

# hdg_1
# hdg_2
# hdg_3
# amfora_link: A link that Amfora supports viewing. For now this is only gemini://
# foreign_link: HTTP(S), Gopher, etc
# link_number: The silver number that appears to the left of a link
# regular_text: Normal gemini text, and plaintext documents
# quote_text
# preformatted_text
# list_text

# btn_bg: The bg color for all modal buttons
# btn_text: The text color for all modal buttons

# dl_choice_modal_bg
# dl_choice_modal_text
# dl_modal_bg
# dl_modal_text
# info_modal_bg
# info_modal_text
# error_modal_bg
# error_modal_text
# yesno_modal_bg
# yesno_modal_text
# tofu_modal_bg
# tofu_modal_text
# subscription_modal_bg
# subscription_modal_text

# input_modal_bg
# input_modal_text
# input_modal_field_bg: The bg of the input field, where you type the text
# input_modal_field_text: The color of the text you type

# bkmk_modal_bg
# bkmk_modal_text
# bkmk_modal_label
# bkmk_modal_field_bg
# bkmk_modal_field_text
`)
