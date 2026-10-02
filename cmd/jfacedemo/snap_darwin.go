package main

import (
	"fmt"
	"structs"

	"github.com/ebitengine/purego/objc"
)

// Window capture is per platform: main calls snapshot, each OS provides it.
type nsRect struct {
	_          structs.HostLayout
	X, Y, W, H float64
}

func str(s string) objc.ID {
	return objc.ID(objc.GetClass("NSString")).Send(objc.RegisterName("stringWithUTF8String:"), s)
}

// snapshot renders the Shell via cacheDisplayInRect:toBitmapImageRep: - screencapture needs
// Screen Recording permission this machine does not grant.
func snapshot(path string) {
	app := objc.ID(objc.GetClass("NSApplication")).Send(objc.RegisterName("sharedApplication"))
	win := shellWindow(app)
	view := win.Send(objc.RegisterName("contentView")).Send(objc.RegisterName("superview"))
	bounds := objc.Send[nsRect](view, objc.RegisterName("bounds"))
	rep := view.Send(objc.RegisterName("bitmapImageRepForCachingDisplayInRect:"), bounds)
	view.Send(objc.RegisterName("cacheDisplayInRect:toBitmapImageRep:"), bounds, rep)
	dict := objc.ID(objc.GetClass("NSDictionary")).Send(objc.RegisterName("dictionary"))
	data := rep.Send(objc.RegisterName("representationUsingType:properties:"), 4, dict)
	ok := objc.Send[bool](data, objc.RegisterName("writeToFile:atomically:"), str(path), true)
	fmt.Println("snapshot written:", ok)
}

// The SWTWindow among NSApp's windows (keyWindow depends on whether the app got activated).
func shellWindow(app objc.ID) objc.ID {
	wins := app.Send(objc.RegisterName("windows"))
	n := objc.Send[int](wins, objc.RegisterName("count"))
	for i := 0; i < n; i++ {
		w := wins.Send(objc.RegisterName("objectAtIndex:"), i)
		if objc.Send[bool](w, objc.RegisterName("isKindOfClass:"), objc.GetClass("SWTWindow")) {
			return w
		}
	}
	panic("no SWTWindow")
}
