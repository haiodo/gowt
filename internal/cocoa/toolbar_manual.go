//go:build darwin

// Hand-written: a real NSToolbar on a window (unified style, Liquid Glass items on macOS 26, search
// field, segmented groups, a spacer that keeps items past a sidebar),
// NSGlassEffectContainerView grouping of glass surfaces. NSToolbar needs a delegate, so one ObjC class is registered at run time.
package cocoa

import (
	"fmt"
	"os"
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
	Sidebar  bool // with SetSidebarEdge: placed over the sidebar, ending at its edge
	Search   func(text string)
	ID       string // names a search item for the QA driver
	Disabled bool // a push item or group segment shown dimmed; SetToolbarItemEnabled changes it
}

const (
	toolbarStyleUnified = 3
	itemIDPrefix        = "gowt.item."
	flexibleSpaceID     = "NSToolbarFlexibleSpaceItem"
	sidebarSpacerID     = "gowt.sidebar"
	sidebarEndID        = "gowt.sidebar.end"
	selectMomentary     = 2
	representExpanded   = 1
	displayIconOnly     = 2
	searchButtonPrefix  = "gowt.searchbutton."
	searchFieldWidth    = 220
)

type toolbarState struct {
	items       []ToolbarItem
	sidebar     bool    // the window has SetSidebarEdge: the spacer item goes first
	collapsed   bool    // the sidebar is hidden (SetSidebarHidden): the spacers are out of the toolbar
	spacer      uintptr // the spacer's view and width constraint, owned by the toolbar item
	width       uintptr
	spacerWidth float64
	end         uintptr         // the spacer after the Sidebar items, the point the spacer width is fitted on
	groups      map[int]uintptr // index in items to its NSToolbarItemGroup, owned by the toolbar
	// index in items of a search item to its magnifier button and its NSSearchToolbarItem, owned by the toolbar
	searchButtons, searchItems map[int]uintptr
	searchFields               map[int]uintptr // index in items to the NSSearchField, owned by its item
}

var (
	toolbarDelegates = map[uintptr]*toolbarState{}
	toolbarSeq       int
	toolbarClass     uintptr
)

func toolbarKey() uintptr { return sel("gowtToolbar") }

// overSidebar reports whether it goes over the sidebar: a Sidebar item of a window with a sidebar.
func (s *toolbarState) overSidebar(it ToolbarItem) bool { return s.sidebar && it.Sidebar && !it.Space }

// ids lists the items in toolbar order. With a sidebar that is the spacer, the Sidebar items, a second
// spacer they end before, then the rest: all navigational up to the title, so AppKit keeps the order.
func (s *toolbarState) ids() uintptr { return s.idList(!s.collapsed) }

// idList is ids without the two spacers when spacers is false, for a hidden sidebar.
func (s *toolbarState) idList(spacers bool) uintptr {
	arr := msg(class("NSMutableArray"), "arrayWithCapacity:", uintptr(len(s.items)+2))
	add := func(i int, it ToolbarItem) {
		id := itemIDPrefix + strconv.Itoa(i)
		if it.Space {
			id = flexibleSpaceID
		}
		if it.Search != nil && collapsibleSearch() {
			msg(arr, "addObject:", nsString(searchButtonPrefix+strconv.Itoa(i)))
		}
		msg(arr, "addObject:", nsString(id))
	}
	if s.sidebar {
		if spacers {
			msg(arr, "addObject:", nsString(sidebarSpacerID))
		}
		over := false
		for i, it := range s.items {
			if s.overSidebar(it) {
				add(i, it)
				over = true
			}
		}
		if over && spacers {
			msg(arr, "addObject:", nsString(sidebarEndID))
		}
	}
	for i, it := range s.items {
		if !s.overSidebar(it) {
			add(i, it)
		}
	}
	return arr
}

// searchOpenAtStart (GOWT_TOOLBAR_SEARCH_OPEN=1) shows search items expanded from the start, for snapshots.
var searchOpenAtStart = os.Getenv("GOWT_TOOLBAR_SEARCH_OPEN") == "1"

// collapsibleSearch reports whether toolbar items can be hidden (macOS 15): a search item is then a
// magnifier button that swaps itself for the field, as in Finder.
func collapsibleSearch() bool {
	return msg(class("NSToolbarItem"), "instancesRespondToSelector:", sel("setHidden:"))&0xff != 0
}

var searchCellClass uintptr

// newSearchField makes an NSSearchField with AppKit's own cell. SWT sets the cell class of every
// NSSearchField to its SWTSearchFieldCell, and a toolbar field with that cell showed an empty capsule.
// The cell brings its own field editor: the window's one is SWT's SWTEditorView, which drops typed
// text for a field that belongs to no SWT widget.
func newSearchField() uintptr {
	if searchCellClass == 0 {
		editor := purego.NewCallback(func(self, _, view uintptr) uintptr { return searchEditor(msg(view, "window")) })
		searchCellClass = allocClass(class("NSSearchFieldCell"), "GowtSearchFieldCell", 0)
		addMethod(searchCellClass, sel("fieldEditorForView:"), editor, "@@:@")
		registerClass(searchCellClass)
	}
	nsf := class("NSSearchField")
	prev := msg(nsf, "cellClass")
	msg(nsf, "setCellClass:", searchCellClass)
	f := msg(msg(nsf, "alloc"), "init")
	msg(nsf, "setCellClass:", prev)
	return f
}

// searchEditor is a plain AppKit field editor kept on window w, one per window like the window's own.
func searchEditor(w uintptr) uintptr {
	if w == 0 {
		return 0
	}
	initAssociated()
	key := sel("gowtSearchEditor")
	if ed := getAssociated(w, key); ed != 0 {
		return ed
	}
	ed := msg(msg(class("NSTextView"), "alloc"), "init")
	msg(ed, "setFieldEditor:", 1)
	msg(ed, "setAllowsUndo:", 1)
	setAssociated(w, key, ed, 1)
	msg(ed, "release")
	return ed
}

// searchButton is the magnifier item standing for the search item number i while it is collapsed.
func (s *toolbarState) searchButton(target uintptr, id string) uintptr {
	i, err := strconv.Atoi(strings.TrimPrefix(id, searchButtonPrefix))
	if err != nil || i < 0 || i >= len(s.items) || s.items[i].Search == nil {
		return 0
	}
	ti := msg(msg(class("NSToolbarItem"), "alloc"), "initWithItemIdentifier:", nsString(id))
	msg(ti, "setTarget:", target)
	msg(ti, "setAction:", sel("gowtSearchOpen:"))
	if img := symbolImage("magnifyingglass", s.items[i].Label); img != 0 {
		msg(ti, "setImage:", img)
	}
	msg(ti, "setLabel:", nsString(s.items[i].Label))
	msg(ti, "setToolTip:", nsString(s.items[i].Label))
	msg(ti, "setHidden:", boolArg(searchOpenAtStart))
	if s.searchButtons == nil {
		s.searchButtons = map[int]uintptr{}
	}
	s.searchButtons[i] = ti
	return msg(ti, "autorelease")
}

// showSearch expands search item i into its field and focuses it, or collapses it into the button.
func (s *toolbarState) showSearch(i int, open bool) {
	b, f := s.searchButtons[i], s.searchItems[i]
	if b == 0 || f == 0 {
		return
	}
	msg(b, "setHidden:", boolArg(open))
	msg(f, "setHidden:", boolArg(!open))
	if open && respondsTo(f, "beginSearchInteraction") {
		// The shown field joins the window only on the toolbar's next layout; focusing it before is a no-op.
		msg(f, "performSelectorOnMainThread:withObject:waitUntilDone:", sel("beginSearchInteraction"), 0, 0)
	}
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
	if id == sidebarSpacerID || id == sidebarEndID {
		ti := msg(msg(class("NSToolbarItem"), "alloc"), "initWithItemIdentifier:", nsString(id))
		v, width := newSpacer()
		msg(ti, "setView:", v)
		msg(v, "release")
		if respondsTo(ti, "setBordered:") {
			msg(ti, "setBordered:", 0)
		}
		msg(ti, "setNavigational:", 1)
		if id == sidebarEndID {
			msgF64(width, sel("setConstant:"), sidebarEndWidth)
			s.end = v
		} else {
			s.spacer, s.width, s.spacerWidth = v, width, -1
		}
		return msg(ti, "autorelease")
	}
	if strings.HasPrefix(id, searchButtonPrefix) {
		return s.searchButton(target, id)
	}
	it := s.find(id)
	if it == nil {
		return 0
	}
	var ti uintptr
	switch {
	case it.Search != nil:
		ti = msg(msg(class("NSSearchToolbarItem"), "alloc"), "initWithItemIdentifier:", nsString(id))
		// The item manages the field once assigned, so the field is configured first.
		f := newSearchField()
		msg(f, "setIdentifier:", nsString(id))
		msg(f, "setPlaceholderString:", nsString(it.Label))
		msg(f, "setTarget:", target)
		msg(f, "setAction:", sel("gowtSearch:"))
		msg(f, "setDelegate:", target)
		msg(ti, "setSearchField:", f)
		msg(f, "release")
		if s.searchFields == nil {
			s.searchFields = map[int]uintptr{}
		}
		fi, _ := strconv.Atoi(strings.TrimPrefix(id, itemIDPrefix))
		s.searchFields[fi] = f
		if respondsTo(ti, "setPreferredWidthForSearchField:") {
			if msgF64 == nil {
				purego.RegisterFunc(&msgF64, objcMsgSend)
			}
			msgF64(ti, sel("setPreferredWidthForSearchField:"), searchFieldWidth)
		}
		if collapsibleSearch() {
			msg(ti, "setHidden:", boolArg(!searchOpenAtStart))
			if s.searchItems == nil {
				s.searchItems = map[int]uintptr{}
			}
			i, _ := strconv.Atoi(strings.TrimPrefix(id, itemIDPrefix))
			s.searchItems[i] = ti
		}
		ti = msg(ti, "autorelease")
	case len(it.Group) > 0:
		ti = groupItem(target, id, it.Group)
		if s.groups == nil {
			s.groups = map[int]uintptr{}
		}
		i, _ := strconv.Atoi(strings.TrimPrefix(id, itemIDPrefix))
		s.groups[i] = ti
		// AppKit validates the group as a whole and enables every segment again; the segments follow
		// their own state instead.
		msg(ti, "setAutovalidates:", 0)
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
	if (it.Nav || s.overSidebar(*it)) && respondsTo(ti, "setNavigational:") {
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
		sub := msg(subs, "objectAtIndex:", uintptr(i))
		msg(sub, "setAutovalidates:", 0)
		msg(sub, "setEnabled:", boolArg(on))
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
	// The spacers stay allowed while the sidebar is hidden, so SetSidebarHidden can insert them again.
	allowed := purego.NewCallback(func(self, _, tb uintptr) uintptr {
		if s := toolbarDelegates[self]; s != nil {
			return s.idList(true)
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
	open := purego.NewCallback(func(self, _, sender uintptr) {
		defer enterCallback()()
		if s := toolbarDelegates[self]; s != nil {
			i, err := strconv.Atoi(strings.TrimPrefix(goString(msg(sender, "itemIdentifier")), searchButtonPrefix))
			if err == nil {
				s.showSearch(i, true)
			}
		}
	})
	// The field collapses into the button when editing ends with no text: Esc, the clear button
	// (it resigns focus by default) or a click elsewhere.
	endEditing := purego.NewCallback(func(self, _, note uintptr) {
		defer enterCallback()()
		s := toolbarDelegates[self]
		field := msg(note, "object")
		if s == nil || field == 0 || goString(msg(field, "stringValue")) != "" {
			return
		}
		if i, err := strconv.Atoi(strings.TrimPrefix(goString(msg(field, "identifier")), itemIDPrefix)); err == nil {
			s.showSearch(i, false)
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
		"toolbarAllowedItemIdentifiers:":                           {allowed, "@@:@"},
		"toolbar:itemForItemIdentifier:willBeInsertedIntoToolbar:": {forID, "@@:@@c"},
		"gowtClick:":                {click, "v@:@"},
		"gowtSearch:":               {search, "v@:@"},
		"validateToolbarItem:":      {validate, "c@:@"},
		"gowtSearchOpen:":           {open, "v@:@"},
		"controlTextDidEndEditing:": {endEditing, "v@:@"},
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
	toolbarDelegates[d] = &toolbarState{items: append([]ToolbarItem(nil), items...), sidebar: sidebar, collapsed: sidebarHidden[w]}
	setAssociated(w, toolbarKey(), d, 1)
	msg(d, "release")
	toolbarSeq++
	tb := msg(msg(class("NSToolbar"), "alloc"), "initWithIdentifier:", nsString(fmt.Sprintf("gowt.toolbar.%d", toolbarSeq)))
	msg(tb, "setDelegate:", d)
	msg(tb, "setDisplayMode:", displayIconOnly) // Finder shows no labels under the items
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
		return // validation would enable the whole group again
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

// ToolbarSearchField is the first search item of a window's toolbar, for FocusSearch and the QA driver.
type ToolbarSearchField struct {
	s *toolbarState
	i int
}

// ToolbarSearch returns the first search item of the toolbar of the window of view; nil when it has none.
func ToolbarSearch(view *NSView) *ToolbarSearchField {
	initAssociated()
	w := msg(uintptr(view.Id), "window")
	if w == 0 {
		return nil
	}
	s := toolbarDelegates[getAssociated(w, toolbarKey())]
	if s == nil {
		return nil
	}
	for i, it := range s.items {
		if it.Search != nil {
			return &ToolbarSearchField{s, i}
		}
	}
	return nil
}

func (f *ToolbarSearchField) field() uintptr { return f.s.searchFields[f.i] }

// ID is the item's ID, "" if it has none.
func (f *ToolbarSearchField) ID() string { return f.s.items[f.i].ID }

// Placeholder is the item's title.
func (f *ToolbarSearchField) Placeholder() string { return f.s.items[f.i].Label }

// Text is what the field holds.
func (f *ToolbarSearchField) Text() string {
	if sf := f.field(); sf != 0 {
		return goString(msg(sf, "stringValue"))
	}
	return ""
}

// SetText replaces the text without running the search action, expanding a collapsed field.
func (f *ToolbarSearchField) SetText(t string) {
	if sf := f.field(); sf != 0 {
		f.s.showSearch(f.i, true)
		msg(sf, "setStringValue:", nsString(t))
	}
}

// Focus expands a collapsed field and makes it the first responder; a field that has only just been
// shown takes the focus on the toolbar's next layout.
func (f *ToolbarSearchField) Focus() bool {
	sf := f.field()
	if sf == 0 {
		return false
	}
	if f.s.searchButtons[f.i] != 0 && f.s.searchItems[f.i] != 0 {
		f.s.showSearch(f.i, true)
		return true
	}
	w := msg(sf, "window")
	return w != 0 && msg(w, "makeFirstResponder:", sf) != 0
}

// Focused reports whether the field or its field editor is the first responder.
func (f *ToolbarSearchField) Focused() bool {
	sf := f.field()
	if sf == 0 {
		return false
	}
	w := msg(sf, "window")
	if w == 0 {
		return false
	}
	r := msg(w, "firstResponder")
	return r == sf || r != 0 && respondsTo(r, "delegate") && msg(r, "delegate") == sf
}

// Submit runs the search with the text in the field, as Return does.
func (f *ToolbarSearchField) Submit() {
	if fn := f.s.items[f.i].Search; fn != nil {
		fn(f.Text())
	}
}

// Collapse folds an empty field back into its magnifier button, as Esc does.
func (f *ToolbarSearchField) Collapse() { f.s.showSearch(f.i, false) }
