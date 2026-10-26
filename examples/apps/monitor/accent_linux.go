package main

import (
	"os/exec"
	"strings"
)

// gnomeAccents are the accent names GNOME 47 and later offers, with the Adwaita colours.
var gnomeAccents = map[string][3]uint8{
	"blue": {53, 132, 228}, "teal": {33, 144, 164}, "green": {58, 148, 74}, "yellow": {200, 136, 0},
	"orange": {237, 91, 0}, "red": {230, 45, 66}, "pink": {213, 97, 153}, "purple": {145, 65, 172},
	"slate": {111, 131, 150},
}

// systemAccent reads the GNOME accent through gsettings; a desktop without the key (older GNOME,
// KDE) gives ok false and the caller keeps its default.
func systemAccent() (r, g, b uint8, ok bool) {
	out, err := exec.Command("gsettings", "get", "org.gnome.desktop.interface", "accent-color").Output()
	if err != nil {
		return 0, 0, 0, false
	}
	c, found := gnomeAccents[strings.Trim(strings.TrimSpace(string(out)), "'")]
	return c[0], c[1], c[2], found
}
