# dmgbuild layout for the Sayki installer window. Run via `make dmg`, which passes
# -D app=<path to Sayki.app> -D background=<dmg-background.png> -D icon=<icon.icns>.
# The background art (dmg-background.swift) is drawn for exactly this geometry.

files = [defines["app"]]
symlinks = {"Applications": "/Applications"}
icon = defines["icon"]  # volume icon shown while mounted
format = "UDZO"

background = defines["background"]  # dmgbuild picks up the @2x file for Retina
window_rect = ((200, 120), (660, 400))
default_view = "icon-view"
show_status_bar = False
show_tab_view = False
show_toolbar = False
show_pathbar = False
show_sidebar = False

icon_size = 128
text_size = 13
icon_locations = {"Sayki.app": (170, 150), "Applications": (490, 150)}
hide_extension = ["Sayki.app"]
