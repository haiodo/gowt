//go:build darwin

// Hand-written: a real NSToolbar on a window (unified style, Liquid Glass items on macOS 26),
// NSGlassEffectContainerView grouping of glass surfaces and NSBackgroundExtensionView around the
// content view. NSToolbar needs a delegate, so one ObjC class is registered at run time.
package cocoa

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/ebitengine/purego"
)

// ToolbarItem is one toolbar entry. Space makes it a flexible space and ignores the other fields.
type ToolbarItem struct {
	Label  string
	Symbol string // SF Symbol name, "" for text only
	Space  bool
	Click  func()
}

const (
	toolbarStyleUnified = 3
	itemIDPrefix        = "gowt.item."
	flexibleSpaceID     = "NSToolbarFlexibleSpaceItem"
)

type toolbarState struct{ items []ToolbarItem }

var (
	toolbarDelegates = map[uintptr]*toolbarState{}
	toolbarSeq       int
	toolbarClass     uintptr
)

func toolbarKey() uintptr { return sel("gowtToolbar") }

func (s *toolbarState) ids() uintptr {
	arr := msg(class("NSMutableArray"), "arrayWithCapacity:", uintptr(len(s.items)))
	for i, it := range s.items {
		id := itemIDPrefix + strconv.Itoa(i)
		if it.Space {
			id = flexibleSpaceID
		}
		msg(arr, "addObject:", nsString(id))
	}
	return arr
}

func (s *toolbarState) item(target uintptr, id string) uintptr {
	i, err := strconv.Atoi(strings.TrimPrefix(id, itemIDPrefix))
	if err != nil || i < 0 || i >= len(s.items) {
		return 0
	}
	it := s.items[i]
	ti := msg(msg(class("NSToolbarItem"), "alloc"), "initWithItemIdentifier:", nsString(id))
	msg(ti, "setTarget:", target)
	msg(ti, "setAction:", sel("gowtClick:"))
	msg(ti, "setLabel:", nsString(it.Label))
	msg(ti, "setToolTip:", nsString(it.Label))
	if it.Symbol != "" {
		img := msg(class("NSImage"), "imageWithSystemSymbolName:accessibilityDescription:", nsString(it.Symbol), nsString(it.Label))
		if img != 0 {
			msg(ti, "setImage:", img)
		}
	}
	return msg(ti, "autorelease")
}

func registerToolbarClass() {
	if toolbarClass != 0 {
		return
	}
	ids := purego.NewCallback(func(self, _, tb uintptr) uintptr {
		if s := toolbarDelegates[self]; s != nil {
			return s.ids()
		}
		return 0
	})
	forID := purego.NewCallback(func(self, _, tb, id, _ uintptr) uintptr {
		if s := toolbarDelegates[self]; s != nil {
			return s.item(self, goString(id))
		}
		return 0
	})
	click := purego.NewCallback(func(self, _, sender uintptr) {
		defer enterCallback()()
		s := toolbarDelegates[self]
		if s == nil {
			return
		}
		i, err := strconv.Atoi(strings.TrimPrefix(goString(msg(sender, "itemIdentifier")), itemIDPrefix))
		if err == nil && i >= 0 && i < len(s.items) && s.items[i].Click != nil {
			s.items[i].Click()
		}
	})
	toolbarClass = newClass("GowtToolbarDelegate", []string{"NSToolbarDelegate"}, map[string]objcMethod{
		"toolbarDefaultItemIdentifiers:":                           {ids, "@@:@"},
		"toolbarAllowedItemIdentifiers:":                           {ids, "@@:@"},
		"toolbar:itemForItemIdentifier:willBeInsertedIntoToolbar:": {forID, "@@:@@c"},
		"gowtClick:": {click, "v@:@"},
	})
}

// InstallToolbar sets an NSToolbar with the unified style on the window of view, replacing an earlier
// one; no items removes it. The items run Click on the UI thread. It reports whether a toolbar is set.
func InstallToolbar(view *NSView, items []ToolbarItem) bool {
	initAssociated()
	w := msg(uintptr(view.Id), "window")
	if w == 0 {
		return false
	}
	if old := getAssociated(w, toolbarKey()); old != 0 {
		delete(toolbarDelegates, old)
		setAssociated(w, toolbarKey(), 0, 1)
	}
	if len(items) == 0 {
		msg(w, "setToolbar:", 0)
		return false
	}
	registerToolbarClass()
	d := msg(msg(toolbarClass, "alloc"), "init")
	toolbarDelegates[d] = &toolbarState{items: append([]ToolbarItem(nil), items...)}
	setAssociated(w, toolbarKey(), d, 1)
	msg(d, "release")
	toolbarSeq++
	tb := msg(msg(class("NSToolbar"), "alloc"), "initWithIdentifier:", nsString(fmt.Sprintf("gowt.toolbar.%d", toolbarSeq)))
	msg(tb, "setDelegate:", d)
	if respondsTo(w, "setToolbarStyle:") {
		msg(w, "setToolbarStyle:", toolbarStyleUnified)
	}
	msg(w, "setToolbar:", tb)
	msg(tb, "release")
	return true
}
