package gowt

import (
	"fmt"
	"os"
	"regexp"
	"sync"

	"github.com/haiodo/gowt/svg"
	"github.com/haiodo/gowt/swt"
)

var (
	iconChain     []string
	iconChainOnce sync.Once
	iconPathsMu   sync.Mutex
	iconPaths     = map[string]string{}
	hexColor      = regexp.MustCompile(`#[0-9a-fA-F]{6}\b`)
)

// systemIcon draws the freedesktop symbolic icon straight from the theme's SVG with the pure Go
// rasterizer; no GTK call is needed. Returns nil when the theme has no such icon.
func systemIcon(a *App, e iconEntry, px int, c RGB) *swt.ImageData {
	iconChainOnce.Do(func() { iconChain = themeChain(iconDirs(), gsettingsTheme()) })
	iconPathsMu.Lock()
	path, ok := iconPaths[e.fd]
	if !ok {
		path = findSymbolic(iconDirs(), iconChain, e.fd)
		iconPaths[e.fd] = path
	}
	iconPathsMu.Unlock()
	if path == "" {
		return nil
	}
	src, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	hex := fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B)
	src = hexColor.ReplaceAll(src, []byte(hex))
	return svg.NewImageDataProvider(src, int32(px), int32(px), hex).GetImageData(100)
}
