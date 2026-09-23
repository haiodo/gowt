// Hand-written: dlopen's the system libraries the translated OS natives resolve against - no
// JNI/Library.loadLibrary equivalent exists in Go (see tooling/j2go/manual.txt). Every native
// binds its own C symbol lazily via purego.Dlsym(purego.RTLD_DEFAULT, ...) on its first call
// (see cocoa_os.go and tooling/j2go/README.md "Native methods"), calling ensureFrameworks()
// first; RTLD_DEFAULT searches every already-loaded image, which is why the frameworks only
// need to be open, not individually handed to each native.
//
// Deliberately NOT a plain package func init(): Go runs same-package func init()s in the
// compiler's file order (lexical filename order for `go build`/`go test`), and a GENERATED
// file's own deferred-static-field func init() (cocoa_os.go, alphabetically before this file)
// would then run before this one dlopen'd anything - class_x fields would objc_getClass against
// nothing yet loaded and silently come back 0. sync.Once makes it independent of file order:
// triggered by the first actual native call, from inside its own lazy-binding closure.
package cocoa

import (
	"sync"

	"github.com/ebitengine/purego"
)

var ensureFrameworksOnce sync.Once

func ensureFrameworks() {
	ensureFrameworksOnce.Do(func() {
		for _, path := range []string{
			"/usr/lib/libobjc.A.dylib",
			"/System/Library/Frameworks/Foundation.framework/Foundation",
			"/System/Library/Frameworks/AppKit.framework/AppKit",
			"/System/Library/Frameworks/CoreFoundation.framework/CoreFoundation",
			"/System/Library/Frameworks/CoreGraphics.framework/CoreGraphics",
			"/System/Library/Frameworks/CoreText.framework/CoreText",
			"/System/Library/Frameworks/QuartzCore.framework/QuartzCore",
		} {
			if _, err := purego.Dlopen(path, purego.RTLD_LAZY|purego.RTLD_GLOBAL); err != nil {
				panic("gowt/internal/cocoa: dlopen " + path + ": " + err.Error())
			}
		}
	})
}
