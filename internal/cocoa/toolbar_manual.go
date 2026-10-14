//go:build darwin

// Hand-written: a real NSToolbar on a window (unified style, Liquid Glass items on macOS 26, search
// field, segmented groups, a separator that tracks a sidebar),
// NSGlassEffectContainerView grouping of glass surfaces. NSToolbar needs a delegate, so one ObjC class is registered at run time.
package cocoa

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/ebitengine/purego"
)

// ToolbarItem is one toolbar entry. Space makes it a flexible space and ignores the other fields;
// Search makes it a search field with Label as placeholder; Group makes it one segmented capsule of
// the group's items, each with its own Label, Symbol and Click.
type ToolbarItem struct {
	Label  string
	Symbol string // SF Symbol name, "" for text only
	Space  bool
	Click  func()
	Group  []ToolbarItem
	Nav    bool // navigational: placed before the window title
	Search func(text string)
}

const (
	toolbarStyleUnified = 3
	itemIDPrefix        = "gowt.item."
	flexibleSpaceID     = "NSToolbarFlexibleSpaceItem"
	sidebarSeparatorID  = "gowt.sidebar"
	selectMomentary     = 2
	representExpanded   = 1
)

type toolbarState struct {
	items []ToolbarItem
	split uintptr // NSSplitView of InstallSidebarTracker, 0 without a sidebar
}

var (
	toolbarDelegates = map[uintptr]*toolbarState{}
	toolbarSeq       int
	toolbarClass     uintptr
)

func toolbarKey() uintptr { return sel("gowtToolbar") }

func (s *toolbarState) ids() uintptr {
	arr := msg(class("NSMutableArray"), "arrayWithCapacity:", uintptr(len(s.items)+1))
	if s.split != 0 {
		msg(arr, "addObject:", nsString(sidebarSeparatorID))
	}
	for i, it := range s.items {
		id := itemIDPrefix + strconv.Itoa(i)
		if it.Space {
			id = flexibleSpaceID
		}
		msg(arr, "addObject:", nsString(id))
	}
	return arr
}

// find returns the item with identifier id, nil for an identifier that is not one of ours.
func (s *toolbarState) find(id string) *ToolbarItem {
	i, err := strconv.Atoi(strings.TrimPrefix(id, itemIDPrefix))
	if err != nil || i < 0 || i >= len(s.items) {
		return nil
	}
	return &s.items[i]
}

func symbolImage(name, label string) uintptr {
	if name == "" {
		return 0
	}
	return msg(class("NSImage"), "imageWithSystemSymbolName:accessibilityDescription:", nsString(name), nsString(label))
}

func (s *toolbarState) item(target uintptr, id string) uintptr {
	if id == sidebarSeparatorID {
		return msg(class("NSTrackingSeparatorToolbarItem"), "trackingSeparatorToolbarItemWithIdentifier:splitView:dividerIndex:", nsString(id), s.split, 0)
	}
	it := s.find(id)
	if it == nil {
		return 0
	}
	var ti uintptr
	switch {
	case it.Search != nil:
		ti = msg(msg(class("NSSearchToolbarItem"), "alloc"), "initWithItemIdentifier:", nsString(id))
		f := msg(ti, "searchField")
		msg(f, "setIdentifier:", nsString(id))
		msg(f, "setPlaceholderString:", nsString(it.Label))
		msg(f, "setTarget:", target)
		msg(f, "setAction:", sel("gowtSearch:"))
		ti = msg(ti, "autorelease")
	case len(it.Group) > 0:
		ti = groupItem(target, id, it.Group)
	default:
		ti = msg(msg(class("NSToolbarItem"), "alloc"), "initWithItemIdentifier:", nsString(id))
		msg(ti, "setTarget:", target)
		msg(ti, "setAction:", sel("gowtClick:"))
		if img := symbolImage(it.Symbol, it.Label); img != 0 {
			msg(ti, "setImage:", img)
		}
		ti = msg(ti, "autorelease")
	}
	msg(ti, "setLabel:", nsString(it.Label))
	msg(ti, "setToolTip:", nsString(it.Label))
	if it.Nav && respondsTo(ti, "setNavigational:") {
		msg(ti, "setNavigational:", 1)
	}
	return ti
}

// groupItem is one momentary segmented capsule, the way Finder shows back and forward. It falls back
// to text segments when a symbol does not exist, since an NSArray cannot hold the missing image.
func groupItem(target uintptr, id string, sub []ToolbarItem) uintptr {
	images := msg(class("NSMutableArray"), "arrayWithCapacity:", uintptr(len(sub)))
	titles := msg(class("NSMutableArray"), "arrayWithCapacity:", uintptr(len(sub)))
	allImages := true
	for _, g := range sub {
		msg(titles, "addObject:", nsString(g.Label))
		img := symbolImage(g.Symbol, g.Label)
		if img == 0 {
			allImages = false
			continue
		}
		msg(images, "addObject:", img)
	}
	var ti uintptr
	if allImages {
		ti = msg(class("NSToolbarItemGroup"), "groupWithItemIdentifier:images:selectionMode:labels:target:action:",
			nsString(id), images, selectMomentary, titles, target, sel("gowtClick:"))
	} else {
		ti = msg(class("NSToolbarItemGroup"), "groupWithItemIdentifier:titles:selectionMode:labels:target:action:",
			nsString(id), titles, selectMomentary, titles, target, sel("gowtClick:"))
	}
	msg(ti, "setControlRepresentation:", representExpanded)
	return ti
}

// clicked runs the handler for a press on the item with identifier id; selected is the segment
// pressed in a group item (selectedIndex), ignored for other items.
func (s *toolbarState) clicked(id string, selected int) {
	it := s.find(id)
	switch {
	case it == nil:
	case len(it.Group) > 0:
		if selected >= 0 && selected < len(it.Group) && it.Group[selected].Click != nil {
			it.Group[selected].Click()
		}
	case it.Click != nil:
		it.Click()
	}
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
		if s == nil || !respondsTo(sender, "itemIdentifier") {
			return
		}
		selected := -1
		if respondsTo(sender, "selectedIndex") {
			selected = int(int64(msg(sender, "selectedIndex")))
		}
		s.clicked(goString(msg(sender, "itemIdentifier")), selected)
	})
	search := purego.NewCallback(func(self, _, field uintptr) {
		defer enterCallback()()
		if s := toolbarDelegates[self]; s != nil {
			if it := s.find(goString(msg(field, "identifier"))); it != nil && it.Search != nil {
				it.Search(goString(msg(field, "stringValue")))
			}
		}
	})
	toolbarClass = newClass("GowtToolbarDelegate", []string{"NSToolbarDelegate"}, map[string]objcMethod{
		"toolbarDefaultItemIdentifiers:":                           {ids, "@@:@"},
		"toolbarAllowedItemIdentifiers:":                           {ids, "@@:@"},
		"toolbar:itemForItemIdentifier:willBeInsertedIntoToolbar:": {forID, "@@:@@c"},
		"gowtClick:":  {click, "v@:@"},
		"gowtSearch:": {search, "v@:@"},
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
	toolbarDelegates[d] = &toolbarState{items: append([]ToolbarItem(nil), items...), split: getAssociated(w, splitKey())}
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

// ReleaseToolbar drops the Go state of the toolbar delegate of the window of view; call it when the
// window is disposed.
func ReleaseToolbar(view *NSView) {
	initAssociated()
	if w := msg(uintptr(view.Id), "window"); w != 0 {
		if d := getAssociated(w, toolbarKey()); d != 0 {
			delete(toolbarDelegates, d)
		}
	}
}
