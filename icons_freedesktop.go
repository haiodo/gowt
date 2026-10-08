package gowt

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// iconDirs are the freedesktop icon base directories, most specific first.
func iconDirs() []string {
	home, _ := os.UserHomeDir()
	data := os.Getenv("XDG_DATA_HOME")
	if data == "" {
		data = filepath.Join(home, ".local", "share")
	}
	dirs := []string{filepath.Join(data, "icons"), filepath.Join(home, ".icons")}
	sys := os.Getenv("XDG_DATA_DIRS")
	if sys == "" {
		sys = "/usr/local/share:/usr/share"
	}
	for _, d := range strings.Split(sys, ":") {
		dirs = append(dirs, filepath.Join(d, "icons"))
	}
	return dirs
}

// themeParents reads Inherits= from <themeDir>/index.theme.
func themeParents(themeDir string) []string {
	f, err := os.Open(filepath.Join(themeDir, "index.theme"))
	if err != nil {
		return nil
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if v, ok := strings.CutPrefix(sc.Text(), "Inherits="); ok {
			return strings.Split(v, ",")
		}
	}
	return nil
}

// themeChain is the theme, its ancestors, then hicolor, as the spec's lookup order.
func themeChain(dirs []string, theme string) []string {
	chain := []string{}
	seen := map[string]bool{}
	var walk func(string)
	walk = func(t string) {
		if t == "" || seen[t] {
			return
		}
		seen[t] = true
		chain = append(chain, t)
		for _, d := range dirs {
			for _, p := range themeParents(filepath.Join(d, t)) {
				walk(strings.TrimSpace(p))
			}
		}
	}
	walk(theme)
	walk("Adwaita")
	walk("hicolor")
	return chain
}

// findSymbolic returns the first "<name>-symbolic.svg" under any size/category directory of the
// themes in chain; "" when there is none.
func findSymbolic(dirs, chain []string, name string) string {
	for _, t := range chain {
		for _, d := range dirs {
			if m, _ := filepath.Glob(filepath.Join(d, t, "*", "*", name+"-symbolic.svg")); len(m) > 0 {
				return m[0]
			}
		}
	}
	return ""
}

// gsettingsTheme reads the GNOME icon theme name; other desktops fall back to Adwaita.
// Ceiling: KDE and Xfce keep theirs elsewhere (kdeglobals, xfconf).
func gsettingsTheme() string {
	out, err := exec.Command("gsettings", "get", "org.gnome.desktop.interface", "icon-theme").Output()
	if err != nil {
		return "Adwaita"
	}
	return strings.Trim(strings.TrimSpace(string(out)), "'")
}
