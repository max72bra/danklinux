package config

// GhosttyConfig contains the default Ghostty configuration
const GhosttyConfig = `# Font Configuration
font-size = 12

# Window Configuration
window-decoration = false
window-padding-x = 12
window-padding-y = 12
background-opacity = 1.0
background-blur-radius = 32

# Cursor Configuration
cursor-style = block
cursor-style-blink = true

# Scrollback
scrollback-limit = 3023

# Terminal features
mouse-hide-while-typing = true
copy-on-select = false
confirm-close-surface = false

# Disable annoying copied to clipboard
app-notifications = no-clipboard-copy,no-config-reload

# Key bindings for common actions
#keybind = ctrl+c=copy_to_clipboard
#keybind = ctrl+v=paste_from_clipboard
keybind = ctrl+shift+n=new_window
keybind = ctrl+t=new_tab
keybind = ctrl+plus=increase_font_size:1
keybind = ctrl+minus=decrease_font_size:1
keybind = ctrl+zero=reset_font_size

# Material 3 UI elements
unfocused-split-opacity = 0.7
unfocused-split-fill = #44464f

# Tab configuration
gtk-titlebar = false

# Shell integration
shell-integration = detect
shell-integration-features = cursor,sudo,title,no-cursor
keybind = shift+enter=text:\n

# Rando stuff
gtk-single-instance = true

# Dank color generation
config-file = ./config-dankcolors
`

// KittyConfig contains the default Kitty configuration
const KittyConfig = `# Font Configuration
font_size 12.0

# Window Configuration
window_padding_width 12
background_opacity 1.0
background_blur 32
hide_window_decorations yes

# Cursor Configuration
cursor_shape block
cursor_blink_interval 1

# Scrollback
scrollback_lines 3000

# Terminal features
copy_on_select yes
strip_trailing_spaces smart

# Key bindings for common actions
map ctrl+shift+n new_window
map ctrl+t new_tab
map ctrl+plus change_font_size all +1.0
map ctrl+minus change_font_size all -1.0
map ctrl+0 change_font_size all 0

# Tab configuration
tab_bar_style powerline
tab_bar_align left

# Shell integration
shell_integration enabled

# Dank color generation
include dank-tabs.conf
include dank-theme.conf
`

const AlacrittyConfig = `[general]
import = [
  "~/.config/alacritty/dank-theme.toml"
]

[window]
decorations = "None"
padding = { x = 12, y = 12 }
opacity = 1.0

[scrolling]
history = 3023

[cursor]
style = { shape = "Block", blinking = "On" }
blink_interval = 500
unfocused_hollow = true

[mouse]
hide_when_typing = true

[selection]
save_to_clipboard = false

[bell]
duration = 0

[keyboard]
bindings = [
  { key = "C",       mods = "Control|Shift", action = "Copy"  },
  { key = "V",       mods = "Control|Shift", action = "Paste" },
  { key = "N",       mods = "Control|Shift", action = "SpawnNewInstance" },
  { key = "Equals",  mods = "Control|Shift", action = "IncreaseFontSize" },
  { key = "Minus",   mods = "Control",       action = "DecreaseFontSize" },
  { key = "Key0",    mods = "Control",       action = "ResetFontSize"    },
  { key = "Enter",   mods = "Shift",         chars = "\n" },
]
`

const AlacrittyThemeConfig = `[colors.primary]
background = '#101418'
foreground = '#e0e2e8'

[colors.selection]
text = '#e0e2e8'
background = '#124a73'

[colors.cursor]
text = '#101418'
cursor = '#9dcbfb'

[colors.normal]
black   = '#101418'
red     = '#d75a59'
green   = '#8ed88c'
yellow  = '#e0d99d'
blue    = '#4087bc'
magenta = '#839fbc'
cyan    = '#9dcbfb'
white   = '#abb2bf'

[colors.bright]
black   = '#5c6370'
red     = '#e57e7e'
green   = '#a2e5a0'
yellow  = '#efe9b3'
blue    = '#a7d9ff'
magenta = '#3d8197'
cyan    = '#5c7ba3'
white   = '#ffffff'
`

const GhosttyColorConfig = `background = #101418
foreground = #e0e2e8
cursor-color = #9dcbfb
selection-background = #124a73
selection-foreground = #e0e2e8
palette = 0=#101418
palette = 1=#d75a59
palette = 2=#8ed88c
palette = 3=#e0d99d
palette = 4=#4087bc
palette = 5=#839fbc
palette = 6=#9dcbfb
palette = 7=#abb2bf
palette = 8=#5c6370
palette = 9=#e57e7e
palette = 10=#a2e5a0
palette = 11=#efe9b3
palette = 12=#a7d9ff
palette = 13=#3d8197
palette = 14=#5c7ba3
palette = 15=#ffffff
`

const KittyThemeConfig = `cursor #e0e2e8
cursor_text_color #c2c7cf

foreground            #e0e2e8
background            #101418
selection_foreground  #243240
selection_background  #b9c8da
url_color             #9dcbfb
color0   #101418
color1   #d75a59
color2   #8ed88c
color3   #e0d99d
color4   #4087bc
color5   #839fbc
color6   #9dcbfb
color7   #abb2bf
color8   #5c6370
color9   #e57e7e
color10   #a2e5a0
color11   #efe9b3
color12   #a7d9ff
color13   #3d8197
color14   #5c7ba3
color15   #ffffff
`

const KittyTabsConfig = `tab_bar_edge            top
tab_bar_style           powerline
tab_powerline_style     slanted
tab_bar_align           left
tab_bar_min_tabs        2
tab_bar_margin_width    0.0
tab_bar_margin_height   2.5 1.5
tab_bar_margin_color    #101418

tab_bar_background              #101418

active_tab_foreground           #cfe5ff
active_tab_background           #124a73
active_tab_font_style           bold

inactive_tab_foreground         #c2c7cf
inactive_tab_background         #101418
inactive_tab_font_style         normal

tab_activity_symbol             " ● "
tab_numbers_style               1

tab_title_template              "{fmt.fg.red}{bell_symbol}{activity_symbol}{fmt.fg.tab}{title[:30]}{title[30:] and '…'} [{index}]"
active_tab_title_template       "{fmt.fg.red}{bell_symbol}{activity_symbol}{fmt.fg.tab}{title[:30]}{title[30:] and '…'} [{index}]"
`
