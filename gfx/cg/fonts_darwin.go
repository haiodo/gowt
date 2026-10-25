//go:build darwin

package cg

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
)

const kCTFontManagerScopeProcess = 1

var (
	cfURLCreateFromFSRep func(alloc uintptr, b *byte, n int, isDir bool) uintptr
	ctRegisterFonts      func(url uintptr, scope int32, errOut *uintptr) bool
	fontsOnce            sync.Once
	fontsMu              sync.Mutex
	fontsDone            = map[string]bool{}
)

// RegisterFonts registers the font files of dir (or the one file) with CoreText for this process only
// and returns how many were new. Call it before the first measuring: earlier lookups keep the fallback.
func RegisterFonts(dir string) (int, error) {
	bind()
	fontsOnce.Do(func() {
		purego.RegisterLibFunc(&cfURLCreateFromFSRep, purego.RTLD_DEFAULT, "CFURLCreateFromFileSystemRepresentation")
		purego.RegisterLibFunc(&ctRegisterFonts, purego.RTLD_DEFAULT, "CTFontManagerRegisterFontsForURL")
	})
	var files []string
	st, err := os.Stat(dir)
	if err != nil {
		return 0, err
	}
	if st.IsDir() {
		ents, err := os.ReadDir(dir)
		if err != nil {
			return 0, err
		}
		for _, e := range ents {
			files = append(files, filepath.Join(dir, e.Name()))
		}
	} else {
		files = []string{dir}
	}
	fontsMu.Lock()
	defer fontsMu.Unlock()
	n := 0
	for _, f := range files {
		switch strings.ToLower(filepath.Ext(f)) {
		case ".otf", ".ttf", ".ttc":
		default:
			continue
		}
		abs, err := filepath.Abs(f)
		if err != nil || fontsDone[abs] {
			continue
		}
		url := cfURLCreateFromFSRep(0, unsafe.StringData(abs), len(abs), false)
		if url == 0 {
			continue
		}
		var cerr uintptr
		ok := ctRegisterFonts(url, kCTFontManagerScopeProcess, &cerr)
		cfRelease(url)
		if cerr != 0 {
			cfRelease(cerr)
		}
		if ok {
			fontsDone[abs] = true
			n++
		}
	}
	return n, nil
}
