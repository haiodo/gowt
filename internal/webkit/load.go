//go:build linux

// Package webkit binds WebKitGTK 4.1 (GTK 3) through purego, without cgo. gen_funcs.go holds the
// prototypes generated from GIR by tooling/webkitgen; the rest is written from the WebKitGTK,
// GObject and Soup reference documentation.
package webkit

import (
	"fmt"
	"strings"
	"sync"

	"github.com/ebitengine/purego"
)

var (
	loadOnce sync.Once
	loadErr  error
	handles  []uintptr
)

// Load opens WebKitGTK and binds every generated function; the error names the missing package.
func Load() error {
	loadOnce.Do(func() {
		for _, n := range []string{"libwebkit2gtk-4.1.so.0", "libjavascriptcoregtk-4.1.so.0", "libsoup-3.0.so.0", "libgtk-3.so.0", "libgobject-2.0.so.0", "libgio-2.0.so.0", "libglib-2.0.so.0"} {
			h, err := purego.Dlopen(n, purego.RTLD_NOW|purego.RTLD_GLOBAL)
			if err != nil {
				loadErr = fmt.Errorf("webview: WebKitGTK 4.1 is not available (%w); install it: apt install libwebkit2gtk-4.1-0 | dnf install webkit2gtk4.1 | pacman -S webkit2gtk-4.1", err)
				return
			}
			handles = append(handles, h)
		}
		if missing := bindAll(); len(missing) > 0 {
			loadErr = fmt.Errorf("webview: WebKitGTK 4.1 lacks %s", strings.Join(missing, ", "))
		}
	})
	return loadErr
}

func lookup(name string) uintptr {
	for _, h := range handles {
		if a, err := purego.Dlsym(h, name); err == nil && a != 0 {
			return a
		}
	}
	return 0
}

func bindOne(fptr any, name string, missing *[]string) {
	a := lookup(name)
	if a == 0 {
		*missing = append(*missing, name)
		return
	}
	purego.RegisterFunc(fptr, a)
}
