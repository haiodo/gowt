//go:build darwin

// Hand-written: a real NSToolbar on a window (unified style, Liquid Glass items on macOS 26, search
// field, segmented groups, a spacer that keeps items past a sidebar),
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
	Label    string
	Symbol   string // SF Symbol name, "" for text only
	Space    bool
	Click    func()
	Group    []ToolbarItem
	Nav      bool // navigational: placed before the window title
	Search   func(text string)
	Disabled bool // a push item or group segment shown dimmed; SetToolbarItemEnabled changes it
}

const (
	toolbarStyleUnified = 3
	itemIDPrefix        = "gowt.item."
	flexibleSpaceID     = "NSToolbarFlexibleSpaceItem"
	sidebarSpacerID     = "gowt.sidebar"
	selectMomentary     = 2
	representExpanded   = 1
)

type toolbarState struct {
	items       []ToolbarItem
	sidebar     bool    // the window has SetSidebarEdge: the spacer item goes first
	spacer      uintptr // the spacer's view and width constraint, owned by the toolbar item
	width       uintptr
	spacerWidth float64
	groups      map[int]uintptr // index in items to its NSToolbarItemGroup, owned by the toolbar
}

var (
	toolbarDelegates = map[uintptr]*toolbarState{}
	toolbarSeq       int
	toolbarClass     uintptr
)

func toolbarKey() uintptr { return sel("gowtToolbar") }

func (s *toolbarState) ids() uintptr {
	arr := msg(class("NSMutableArray"), "arrayWithCapacity:", uintptr(len(s.items)+1))
	if s.sidebar {
		msg(arr, "addObject:", nsString(sidebarSpacerID))
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
	if id == sidebarSpacerID {
		ti := msg(msg(class("NSToolbarItem"), "alloc"), "initWithItemIdentifier:", nsString(id))
		v, width := newSpacer()
		msg(ti, "setView:", v)
		msg(v, "release")
		if respondsTo(ti, "setBordered:") {
			msg(ti, "setBordered:", 0)
		}
		msg(ti, "setNavigational:", 1)
		s.spacer, s.width, s.spacerWidth = v, width, -1
		return msg(ti, "autorelease")
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
		if s.groups == nil {
			s.groups = map[int]uintptr{}
		}
		i, _ := strconv.Atoi(strings.TrimPrefix(id, itemIDPrefix))
		s.groups[i] = ti
		for j, g := range it.Group {
			setSegmentEnabled(ti, j, !g.Disabled)
		}
	default:
		ti = msg(msg(class("NSToolbarItem"), "alloc"), "initWithItemIdentifier:", nsString(id))
		msg(ti, "setTarget:", target)
		msg(ti, "setAction:", sel("gowtClick:"))
		if img := symbolImage(it.Symbol, it.Label); img != 0 {
			msg(ti, "setImage:", img)
		}
		msg(ti, "setEnabled:", boolArg(!it.Disabled))
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

// setSegmentEnabled sets segment i of a group item both on its subitem, which toolbar validation
// reads, and on the segmented control that shows it.
func setSegmentEnabled(group uintptr, i int, on bool) {
	subs := msg(group, "subitems")
	if i < int(msg(subs, "count")) {
		msg(msg(subs, "objectAtIndex:", uintptr(i)), "setEnabled:", boolArg(on))
	}
	if v := msg(group, "view"); respondsTo(v, "setEnabled:forSegment:") {
		msg(v, "setEnabled:forSegment:", boolArg(on), uintptr(i))
	}
}

// enabled answers validateToolbarItem: for item, a push item or a subitem of a group: autovalidation
// would otherwise enable every item whose target answers its action.
func (s *toolbarState) enabled(item uintptr) bool {
	for i, g := range s.groups {
		subs := msg(g, "subitems")
		if j := int(msg(subs, "indexOfObject:", item)); j >= 0 && j < int(msg(subs, "count")) && j < len(s.items[i].Group) {
			return !s.items[i].Group[j].Disabled
		}
	}
	if it := s.find(goString(msg(item, "itemIdentifier"))); it != nil && len(it.Group) == 0 {
		return !it.Disabled
	}
	return true
}

func boolArg(v bool) uintptr {
	if v {
		return 1
	}
	return 0
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
	validate := purego.NewCallback(func(self, _, item uintptr) uintptr {
		if s := toolbarDelegates[self]; s != nil {
			return boolArg(s.enabled(item))
		}
		return 1
	})
	toolbarClass = newClass("GowtToolbarDelegate", []string{"NSToolbarDelegate"}, map[string]objcMethod{
		"toolbarDefaultItemIdentifiers:":                           {ids, "@@:@"},
		"toolbarAllowedItemIdentifiers:":                           {ids, "@@:@"},
		"toolbar:itemForItemIdentifier:willBeInsertedIntoToolbar:": {forID, "@@:@@c"},
		"gowtClick:":           {click, "v@:@"},
		"gowtSearch:":          {search, "v@:@"},
		"validateToolbarItem:": {validate, "c@:@"},
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
	_, sidebar := sidebarEdges[w]
	toolbarDelegates[d] = &toolbarState{items: append([]ToolbarItem(nil), items...), sidebar: sidebar}
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

// SetToolbarItemEnabled enables or dims item number item of the toolbar of the window of view, or
// segment number segment of that group item when segment is not negative. A toolbar rebuilt for a
// sidebar keeps the state, since it is stored in the items.
func SetToolbarItemEnabled(view *NSView, item, segment int, on bool) {
	initAssociated()
	w := msg(uintptr(view.Id), "window")
	if w == 0 {
		return
	}
	s := toolbarDelegates[getAssociated(w, toolbarKey())]
	if s == nil || item < 0 || item >= len(s.items) {
		return
	}
	it := &s.items[item]
	switch {
	case segment < 0:
		it.Disabled = !on
	case segment < len(it.Group):
		it.Group[segment].Disabled = !on
		if g := s.groups[item]; g != 0 {
			setSegmentEnabled(g, segment, on)
		}
	default:
		return
	}
	if tb := msg(w, "toolbar"); tb != 0 {
		msg(tb, "validateVisibleItems")
	}
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
