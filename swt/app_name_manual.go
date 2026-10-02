package swt

import (
	"os"
	"path/filepath"
	"strings"
)

// The generated default is "SWT" (empty on macOS). A process started as myapp shows up as myapp;
// DisplaySetAppName still overrides it.
func init() {
	if DisplayAPP_NAME != "" && DisplayAPP_NAME != "SWT" {
		return
	}
	exe, err := os.Executable()
	// Inside a .app bundle macOS takes the name from CFBundleName.
	if err != nil || strings.Contains(exe, ".app/Contents/") {
		return
	}
	name := filepath.Base(exe)
	if strings.EqualFold(filepath.Ext(name), ".exe") {
		name = name[:len(name)-4]
	}
	DisplayAPP_NAME = name
}
