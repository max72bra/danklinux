package config

const HyprlandConfig = `# Hyprland Configuration
# https://wiki.hypr.land/Configuring/

# ==================
# MONITOR CONFIG
# ==================
# monitor = eDP-2, 2560x1600@239.998993, 2560x0, 1, vrr, 1

# ==================
# ENVIRONMENT VARS
# ==================
env = QT_QPA_PLATFORM,wayland
env = ELECTRON_OZONE_PLATFORM_HINT,auto
env = QT_QPA_PLATFORMTHEME,gtk3
env = QT_QPA_PLATFORMTHEME_QT6,gtk3
env = TERMINAL,{{TERMINAL_COMMAND}}

# ==================
# STARTUP APPS
# ==================
exec-once = bash -c "wl-paste --watch cliphist store &"
exec-once = dms run
exec-once = {{POLKIT_AGENT_PATH}}

# ==================
# INPUT CONFIG
# ==================
input {
    kb_layout = us
    numlock_by_default = true
}

# ==================
# GENERAL LAYOUT
# ==================
general {
    gaps_in = 5
    gaps_out = 5
    border_size = 0  # off in niri
    
    col.active_border = rgba(707070ff)
    col.inactive_border = rgba(d0d0d0ff)
    
    layout = dwindle
}

# ==================
# DECORATION
# ==================
decoration {
    rounding = 12
    
    active_opacity = 1.0
    inactive_opacity = 0.9
    
    shadow {
        enabled = true
        range = 30
        render_power = 5
        offset = 0 5
        color = rgba(00000070)
    }
}

# ==================
# ANIMATIONS
# ==================
animations {
    enabled = true
    
    animation = windowsIn, 1, 3, default
    animation = windowsOut, 1, 3, default
    animation = workspaces, 1, 5, default
    animation = windowsMove, 1, 4, default
    animation = fade, 1, 3, default
    animation = border, 1, 3, default
}

# ==================
# LAYOUTS
# ==================
dwindle {
    preserve_split = true
}

master {
    mfact = 0.5
}

# ==================
# MISC
# ==================
misc {
    disable_hyprland_logo = true
    disable_splash_rendering = true
    vrr = 1
}

# ==================
# WINDOW RULES
# ==================
windowrulev2 = tile, class:^(org\.wezfurlong\.wezterm)$

windowrulev2 = rounding 12, class:^(org\.gnome\.)
windowrulev2 = noborder, class:^(org\.gnome\.)

windowrulev2 = tile, class:^(gnome-control-center)$
windowrulev2 = tile, class:^(pavucontrol)$
windowrulev2 = tile, class:^(nm-connection-editor)$

windowrulev2 = float, class:^(gnome-calculator)$
windowrulev2 = float, class:^(galculator)$
windowrulev2 = float, class:^(blueman-manager)$
windowrulev2 = float, class:^(org\.gnome\.Nautilus)$
windowrulev2 = float, class:^(steam)$
windowrulev2 = float, class:^(xdg-desktop-portal)$

windowrulev2 = noborder, class:^(org\.wezfurlong\.wezterm)$
windowrulev2 = noborder, class:^(Alacritty)$
windowrulev2 = noborder, class:^(zen)$
windowrulev2 = noborder, class:^(com\.mitchellh\.ghostty)$
windowrulev2 = noborder, class:^(kitty)$

windowrulev2 = float, class:^(firefox)$, title:^(Picture-in-Picture)$
windowrulev2 = float, class:^(zoom)$

windowrulev2 = opacity 0.9 0.9, floating:0, focus:0

layerrule = noanim, ^(quickshell)$

# ==================
# KEYBINDINGS
# ==================
$mod = SUPER

# === Application Launchers ===
bind = $mod, T, exec, {{TERMINAL_COMMAND}}
bind = $mod, space, exec, dms ipc call spotlight toggle
bind = $mod, V, exec, dms ipc call clipboard toggle
bind = $mod, M, exec, dms ipc call processlist toggle
bind = $mod, comma, exec, dms ipc call settings toggle
bind = $mod, N, exec, dms ipc call notifications toggle
bind = $mod, SHIFT, N, exec, dms ipc call notepad toggle
bind = $mod, Y, exec, dms ipc call dankdash wallpaper
bind = $mod, TAB, exec, dms ipc call hypr toggleOverview

# === Security ===
bind = $mod ALT, L, exec, dms ipc call lock lock
bind = $mod SHIFT, E, exit
bind = CTRL ALT, Delete, exec, dms ipc call processlist toggle

# === Audio Controls ===
bindel = , XF86AudioRaiseVolume, exec, dms ipc call audio increment 3
bindel = , XF86AudioLowerVolume, exec, dms ipc call audio decrement 3
bindl = , XF86AudioMute, exec, dms ipc call audio mute
bindl = , XF86AudioMicMute, exec, dms ipc call audio micmute

# === Keyboard Backlight ===
bindel = , XF86KbdBrightnessUp, exec, kbdbrite.sh up
bindel = , XF86KbdBrightnessDown, exec, kbdbrite.sh down

# === Brightness Controls ===
bindel = , XF86MonBrightnessUp, exec, dms ipc call brightness increment 5
bindel = , XF86MonBrightnessDown, exec, dms ipc call brightness decrement 5

# === Window Management ===
bind = $mod, Q, killactive
bind = $mod, F, fullscreen, 1
bind = $mod SHIFT, F, fullscreen, 0
bind = $mod SHIFT, T, togglefloating
bind = $mod, W, togglegroup

# === Focus Navigation ===
bind = $mod, left, movefocus, l
bind = $mod, down, movefocus, d
bind = $mod, up, movefocus, u
bind = $mod, right, movefocus, r
bind = $mod, H, movefocus, l
bind = $mod, J, movefocus, d
bind = $mod, K, movefocus, u
bind = $mod, L, movefocus, r

# === Window Movement ===
bind = $mod SHIFT, left, movewindow, l
bind = $mod SHIFT, down, movewindow, d
bind = $mod SHIFT, up, movewindow, u
bind = $mod SHIFT, right, movewindow, r
bind = $mod SHIFT, H, movewindow, l
bind = $mod SHIFT, J, movewindow, d
bind = $mod SHIFT, K, movewindow, u
bind = $mod SHIFT, L, movewindow, r

# === Column Navigation ===
bind = $mod, Home, focuswindow, first
bind = $mod, End, focuswindow, last

# === Monitor Navigation ===
bind = $mod CTRL, left, focusmonitor, l
bind = $mod CTRL, right, focusmonitor, r
bind = $mod CTRL, H, focusmonitor, l
bind = $mod CTRL, J, focusmonitor, d
bind = $mod CTRL, K, focusmonitor, u
bind = $mod CTRL, L, focusmonitor, r

# === Move to Monitor ===
bind = $mod SHIFT CTRL, left, movewindow, mon:l
bind = $mod SHIFT CTRL, down, movewindow, mon:d
bind = $mod SHIFT CTRL, up, movewindow, mon:u
bind = $mod SHIFT CTRL, right, movewindow, mon:r
bind = $mod SHIFT CTRL, H, movewindow, mon:l
bind = $mod SHIFT CTRL, J, movewindow, mon:d
bind = $mod SHIFT CTRL, K, movewindow, mon:u
bind = $mod SHIFT CTRL, L, movewindow, mon:r

# === Workspace Navigation ===
bind = $mod, Page_Down, workspace, e+1
bind = $mod, Page_Up, workspace, e-1
bind = $mod, U, workspace, e+1
bind = $mod, I, workspace, e-1
bind = $mod CTRL, down, movetoworkspace, e+1
bind = $mod CTRL, up, movetoworkspace, e-1
bind = $mod CTRL, U, movetoworkspace, e+1
bind = $mod CTRL, I, movetoworkspace, e-1

# === Move Workspaces ===
bind = $mod SHIFT, Page_Down, movetoworkspace, e+1
bind = $mod SHIFT, Page_Up, movetoworkspace, e-1
bind = $mod SHIFT, U, movetoworkspace, e+1
bind = $mod SHIFT, I, movetoworkspace, e-1

# === Mouse Wheel Navigation ===
bind = $mod, mouse_down, workspace, e+1
bind = $mod, mouse_up, workspace, e-1
bind = $mod CTRL, mouse_down, movetoworkspace, e+1
bind = $mod CTRL, mouse_up, movetoworkspace, e-1

# === Numbered Workspaces ===
bind = $mod, 1, workspace, 1
bind = $mod, 2, workspace, 2
bind = $mod, 3, workspace, 3
bind = $mod, 4, workspace, 4
bind = $mod, 5, workspace, 5
bind = $mod, 6, workspace, 6
bind = $mod, 7, workspace, 7
bind = $mod, 8, workspace, 8
bind = $mod, 9, workspace, 9

# === Move to Numbered Workspaces ===
bind = $mod SHIFT, 1, movetoworkspace, 1
bind = $mod SHIFT, 2, movetoworkspace, 2
bind = $mod SHIFT, 3, movetoworkspace, 3
bind = $mod SHIFT, 4, movetoworkspace, 4
bind = $mod SHIFT, 5, movetoworkspace, 5
bind = $mod SHIFT, 6, movetoworkspace, 6
bind = $mod SHIFT, 7, movetoworkspace, 7
bind = $mod SHIFT, 8, movetoworkspace, 8
bind = $mod SHIFT, 9, movetoworkspace, 9

# === Column Management ===
bind = $mod, bracketleft, layoutmsg, preselect l
bind = $mod, bracketright, layoutmsg, preselect r

# === Sizing & Layout ===
bind = $mod, R, layoutmsg, togglesplit
bind = $mod CTRL, F, resizeactive, exact 100%

# === Move/resize windows with mainMod + LMB/RMB and dragging ===
bindmd = $mod, mouse:272, Move window, movewindow
bindmd = $mod, mouse:273, Resize window, resizewindow

# === Move/resize windows with mainMod + LMB/RMB and dragging ===
bindd = $mod, code:20, Expand window left, resizeactive, -100 0
bindd = $mod, code:21, Shrink window left, resizeactive, 100 0

# === Manual Sizing ===
binde = $mod, minus, resizeactive, -10% 0
binde = $mod, equal, resizeactive, 10% 0
binde = $mod SHIFT, minus, resizeactive, 0 -10%
binde = $mod SHIFT, equal, resizeactive, 0 10%

# === Screenshots ===
bind = , XF86Launch1, exec, grimblast copy area
bind = CTRL, XF86Launch1, exec, grimblast copy screen
bind = ALT, XF86Launch1, exec, grimblast copy active
bind = , Print, exec, grimblast copy area
bind = CTRL, Print, exec, grimblast copy screen
bind = ALT, Print, exec, grimblast copy active

# === System Controls ===
bind = $mod SHIFT, P, dpms, off`
