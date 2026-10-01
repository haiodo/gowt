package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"structs"

	"github.com/ebitengine/purego/objc"
	"github.com/haiodo/gowt/swt"
)

// The -snap driver talks to AppKit, so it is per platform: main calls snapAll, each OS provides it.
// snapAll selects each tab the way a click does (NSTabView selectTabViewItemAtIndex:, which
// runs TabFolder's own delegate path), waits for layout and paint, then snapshots the window.
func snapAll(display *swt.Display, shell *swt.Shell, folder *swt.TabFolder, dir string) {
	snapRun(display, shell, folder, dir, snapHooks{writeMeta, selectTab, clickButton, snapshot})
}

// writeMeta records what the pixels depend on; snapcheck skips the comparison when it differs
// from the references.
func writeMeta(path string) {
	app := objc.ID(objc.GetClass("NSApplication")).Send(objc.RegisterName("sharedApplication"))
	scale := objc.Send[float64](shellWindow(app), objc.RegisterName("backingScaleFactor"))
	version, _ := exec.Command("sw_vers", "-productVersion").Output()
	style, _ := exec.Command("defaults", "read", "-g", "AppleInterfaceStyle").Output() // fails in light mode
	appearance := "light"
	if strings.TrimSpace(string(style)) == "Dark" {
		appearance = "dark"
	}
	meta := fmt.Sprintf("scale=%g\nmacos=%s\nappearance=%s\n", scale, strings.TrimSpace(string(version)), appearance)
	if err := os.WriteFile(path, []byte(meta), 0o644); err != nil {
		panic(err)
	}
}

func selectTab(folder *swt.TabFolder, index int) {
	objc.ID(folder.View.Id).Send(objc.RegisterName("selectTabViewItemAtIndex:"), index)
}

// clickButton finds the Button labelled text under root and sends it performClick:, the real
// NSButton target/action path.
func clickButton(root *swt.Control, text string) {
	b := findButton(root, text)
	if b == nil {
		fmt.Println("button not found:", text)
		return
	}
	objc.ID(b.View.Id).Send(objc.RegisterName("performClick:"), 0)
	fmt.Printf("clicked %s: selection=%v\n", text, b.GetSelection())
}

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
	fmt.Println("snapshot", path, ok)
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
