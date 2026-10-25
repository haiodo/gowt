package gowt

import (
	"runtime"
	"strings"
)

// winBody is the WinUI 3 body size; the Win32 UI font (Segoe UI 9 pt, 12 px) is smaller.
const winBody = 14

// systemInfo reads what the token resolver needs from the running OS.
func (a *App) systemInfo() system {
	s := system{goos: runtime.GOOS}
	fd := a.display.GetSystemFont().GetFontData()
	pt := 13.0
	if len(fd) > 0 {
		pt = float64(fd[0].GetHeightF())
	}
	switch s.goos {
	case "windows":
		s.systemPx = pt * 96 / 72
		s.bodyPx = winBody
		s.family = a.firstInstalled("Segoe UI Variable Text", "Segoe UI")
		if s.family == "Segoe UI Variable Text" {
			s.display = a.firstInstalled("Segoe UI Variable Display", s.family)
		}
		s.mono = a.firstInstalled("Cascadia Mono", "Consolas")
	case "linux":
		s.systemPx = pt * float64(a.display.GetDPI().Y) / 72
		s.bodyPx = s.systemPx
		s.mono = gsettingsMono()
	default:
		s.systemPx, s.bodyPx = pt, pt
		s.mono = "Menlo"
	}
	return s
}

// firstInstalled returns the first of the families the OS has, or the last one.
func (a *App) firstInstalled(families ...string) string {
	for _, f := range families[:len(families)-1] {
		if len(a.display.GetFontList(f, true)) > 0 {
			return f
		}
	}
	return families[len(families)-1]
}

// monoFamily turns "'Ubuntu Sans Mono 13'" into "Ubuntu Sans Mono": the quotes and a trailing size go.
func monoFamily(s string) string {
	s = strings.Trim(strings.TrimSpace(s), "'\"")
	if i := strings.LastIndex(s, " "); i > 0 && strings.Trim(s[i+1:], "0123456789.") == "" {
		s = s[:i]
	}
	if s == "" {
		return "Monospace"
	}
	return s
}
