//go:build darwin

// Hand-written: a real NSToolbar on a window (unified style, Liquid Glass items on macOS 26, search
// field, segmented groups, a spacer that keeps items past a sidebar, a separator that keeps items
// over an inspector), NSGlassEffectContainerView grouping of glass surfaces. NSToolbar needs a delegate, so one ObjC class is registered at run time.
package cocoa

import (
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/ebitengine/purego"
)

// ToolbarItem is one toolbar entry. Space makes it a flexible space and ignores the other fields;
// Search makes it a search field with Label as placeholder; Group makes it one segmented capsule of
// the group's items, each with its own Label, Symbol and Click.
type ToolbarItem struct {
	Label   string
	Symbol  string // SF Symbol name, "" for text only
	Space   bool
	Click   func()
	Group   []ToolbarItem
	Nav     bool // navigational: placed before the window title
	Sidebar bool // with SetSidebarEdge: placed over the sidebar, ending at its edge
	// SidebarEnd places the item just past the right edge of the sidebar, before the window title: a
	// bare icon without the bordered capsule the Sidebar items keep. It follows the edge on a drag or
	// resize and, with the sidebar hidden, sits at the leading edge, next to the traffic lights, as
	// the way back. Without SetSidebarEdge it is an ordinary item.
	SidebarEnd bool
	// Center places the item in the middle of the content column, between the sidebar edge and the
	// inspector divider, fitted as the sidebar spacers are. Without a sidebar or inspector it is the
	// middle of the window.
	Center    bool
	Inspector bool   // with InstallInspectorTracker: after the separator on the inspector divider
	Text      string // a push item shows this as a bordered title instead of its symbol; SetToolbarItemText changes it
	Small     bool   // NSControlSizeSmall: the item gets a button view of that size
	Search    func(text string)
	ID        string // names the item for the QA driver
	Disabled  bool   // a push item or group segment shown dimmed; SetToolbarItemEnabled changes it
}

const (
	toolbarStyleUnified = 3
	itemIDPrefix        = "gowt.item."
	flexibleSpaceID     = "NSToolbarFlexibleSpaceItem"
	sidebarSpacerID     = "gowt.sidebar"
	sidebarEndID        = "gowt.sidebar.end"
	centerSpacerID      = "gowt.center"
	inspectorID         = "gowt.inspector"
	selectMomentary     = 2
	representExpanded   = 1
	displayIconOnly     = 2
	searchButtonPrefix  = "gowt.searchbutton."
	searchFieldWidth    = 220
	// visibilityPriorityLow is NSToolbarItemVisibilityPriorityLow: content items collapse into the
	// overflow chevron before the sidebar cluster, the title and the inspector separator do;
	// visibilityPriorityHigh keeps the Inspector items on the divider through that.
	visibilityPriorityLow  = -1000
	visibilityPriorityHigh = 1000
)

type toolbarState struct {
	items       []ToolbarItem
	sidebar     bool    // the window has SetSidebarEdge: the spacer item goes first
	collapsed   bool    // the sidebar is hidden (SetSidebarHidden): the spacers are out of the toolbar
	spacer      uintptr // the spacer's view and width constraint, owned by the toolbar item
	width       uintptr
	spacerWidth float64
	end         uintptr // the spacer after the Sidebar items, the point the spacer width is fitted on
	center      uintptr // the spacer before the Center items, fitted on the middle of the content column
	centerC     uintptr // the center spacer's width constraint
	centerWidth float64
	centerIdx   map[int]bool    // indexes in items of the Center items, to measure their extent when fitting
	fitTries    int             // fits posted and not yet settled, see postFit
	fitPending  bool            // a posted fit is on the run loop
	groups      map[int]uintptr // index in items to its NSToolbarItemGroup, owned by the toolbar
	// index in items of a search item to its magnifier button and its NSSearchToolbarItem, owned by the toolbar
	searchButtons, searchItems map[int]uintptr
	searchFields               map[int]uintptr // index in items to the NSSearchField, owned by its item
	split                      uintptr         // NSSplitView of InstallInspectorTracker, 0 without an inspector
}

var (
	toolbarDelegates = map[uintptr]*toolbarState{}
	toolbarSeq       int
	toolbarClass     uintptr
)

func toolbarKey() uintptr { return sel("gowtToolbar") }

// overSidebar reports whether it goes over the sidebar: a Sidebar item of a window with a sidebar.
func (s *toolbarState) overSidebar(it ToolbarItem) bool { return s.sidebar && it.Sidebar && !it.Space }

// overSidebarEnd reports whether it follows the sidebar edge: a SidebarEnd item of a window with a
// sidebar. Without a sidebar the item is an ordinary one, as its look.ToolbarItem comment says.
func (s *toolbarState) overSidebarEnd(it ToolbarItem) bool {
	return s.sidebar && it.SidebarEnd && !it.Space
}

// overInspector reports whether it goes after the inspector separator: an Inspector item of a window
// with an inspector.
func (s *toolbarState) overInspector(it ToolbarItem) bool { return s.split != 0 && it.Inspector }

// ids lists the items in toolbar order. With a sidebar that is the spacer, the Sidebar items, a second
// spacer they end before, the SidebarEnd items just past the sidebar edge, then the rest: all
// navigational up to the title, so AppKit keeps the order. The Center items lead the rest, followed
// by the fitted center spacer. With an inspector the Inspector items come last, after the separator
// that tracks its divider.
func (s *toolbarState) ids() uintptr { return s.idList(!s.collapsed) }

// hasCenter reports whether any item goes to the middle of the content column.
func (s *toolbarState) hasCenter() bool {
	for _, it := range s.items {
		if it.Center && !it.Space {
			return true
		}
	}
	return false
}

// idList is ids without the two spacers when spacers is false, for a hidden sidebar.
func (s *toolbarState) idList(spacers bool) uintptr {
	arr := msg(class("NSMutableArray"), "arrayWithCapacity:", uintptr(len(s.items)+3))
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
		for i, it := range s.items {
			if s.overSidebarEnd(it) {
				add(i, it)
			}
		}
	}
	// The Center items lead the trailing block, the fitted spacer goes after them: AppKit packs the
	// non-navigational items against the trailing cluster (up to the inspector separator), so a
	// wider spacer shifts the Center items left, and fitCenter sizes it on that.
	rest := func(center bool) {
		for i, it := range s.items {
			if !s.overSidebar(it) && !s.overSidebarEnd(it) && !s.overInspector(it) && (it.Center && !it.Space) == center {
				add(i, it)
			}
		}
	}
	rest(true)
	if s.hasCenter() {
		msg(arr, "addObject:", nsString(centerSpacerID))
	}
	rest(false)
	sep := false
	for i, it := range s.items {
		if !s.overInspector(it) {
			continue
		}
		if !sep {
			msg(arr, "addObject:", nsString(inspectorID))
			sep = true
		}
		add(i, it)
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
	if id == sidebarSpacerID || id == sidebarEndID || id == centerSpacerID {
		ti := msg(msg(class("NSToolbarItem"), "alloc"), "initWithItemIdentifier:", nsString(id))
		v, width := newSpacer()
		msg(ti, "setView:", v)
		msg(v, "release")
		if respondsTo(ti, "setBordered:") {
			msg(ti, "setBordered:", 0)
		}
		if id != centerSpacerID {
			// The center spacer stays inside the trailing block, between the Center items and the rest.
			msg(ti, "setNavigational:", 1)
		}
		switch id {
		case centerSpacerID:
			s.center, s.centerC, s.centerWidth = v, width, -1
		case sidebarEndID:
			msgF64(width, sel("setConstant:"), sidebarEndWidth)
			s.end = v
		default:
			s.spacer, s.width, s.spacerWidth = v, width, -1
		}
		return msg(ti, "autorelease")
	}
	if id == inspectorID {
		if s.split == 0 {
			return 0
		}
		ti := msg(class("NSTrackingSeparatorToolbarItem"), "trackingSeparatorToolbarItemWithIdentifier:splitView:dividerIndex:", nsString(id), s.split, 0)
		if respondsTo(ti, "setVisibilityPriority:") {
			high := int64(visibilityPriorityHigh)
			msg(ti, "setVisibilityPriority:", uintptr(high))
		}
		return ti
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
		if it.Small {
			setSmall(f)
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
		if v := msg(ti, "view"); it.Small && v != 0 {
			setSmall(v)
		}
	default:
		ti = msg(msg(class("NSToolbarItem"), "alloc"), "initWithItemIdentifier:", nsString(id))
		msg(ti, "setTarget:", target)
		msg(ti, "setAction:", sel("gowtClick:"))
		var img uintptr
		if it.Small {
			img = smallSymbolImage(it.Symbol, it.Label)
		} else {
			img = symbolImage(it.Symbol, it.Label)
		}
		switch {
		case it.Small && (it.Text != "" || it.Symbol == ""):
			// AppKit's own bordered text item exposes no font and has no view to set one on, so a
			// Small text item gets a small toolbar button, as the control size cannot reach it.
			s.smallTextButton(ti, target, id, it)
		case it.Text != "":
			// A title on an item without a custom view gives AppKit's own bordered text button.
			msg(ti, "setTitle:", nsString(it.Text))
			msg(ti, "setBordered:", 1)
		case img != 0 && !it.Nav && !s.overSidebar(*it) && respondsTo(ti, "setBordered:"):
			// AppKit 26+ draws an image item as a bordered glass capsule; the Harness rest state
			// is a bare icon. AppKit draws no hover for such an item, so it gets its own button
			// view, which tracks the pointer in the key window and draws the quiet fill.
			v := bareIconButton(target, id, img)
			msg(ti, "setView:", v)
			msg(v, "release")
			if it.Disabled {
				msg(v, "setEnabled:", 0)
			}
		case img != 0:
			msg(ti, "setImage:", img)
		}
		msg(ti, "setEnabled:", boolArg(!it.Disabled))
		// Content items give way before the sidebar, the title cluster and the inspector separator:
		// on a narrow window AppKit hides the low-priority ones into the overflow chevron and brings
		// them back when the window grows, instead of letting them cross the inspector divider.
		// Inspector items stay: they keep a high priority, above the content ones.
		if respondsTo(ti, "setVisibilityPriority:") {
			low, high := int64(visibilityPriorityLow), int64(visibilityPriorityHigh)
			if !it.Nav && !s.overSidebar(*it) && !s.overSidebarEnd(*it) && !s.overInspector(*it) {
				msg(ti, "setVisibilityPriority:", uintptr(low))
			} else if s.overInspector(*it) {
				msg(ti, "setVisibilityPriority:", uintptr(high))
			}
		}
		ti = msg(ti, "autorelease")
	}
	msg(ti, "setLabel:", nsString(it.Label))
	msg(ti, "setToolTip:", nsString(it.Label))
	if (it.Nav || s.overSidebar(*it) || s.overSidebarEnd(*it)) && respondsTo(ti, "setNavigational:") {
		msg(ti, "setNavigational:", 1)
	}
	return ti
}

const (
	controlSizeSmall = 1  // NSControlSizeSmall
	bezelToolbar     = 11 // NSBezelStyleToolbar
	pushIn           = 7  // NSButtonTypeMomentaryPushIn
)

var (
	smallSymbolOnce sync.Once
	// objc_msgSend with two float arguments needs a typed signature; msg only passes words.
	msgSmallSymbolConfig func(self, sel uintptr, pointSize, weight float64, scale int64) uintptr
)

// smallSymbolImage is the symbol image of a Small item, drawn 12 pt: the item has no control size,
// so the point size is baked into the image; 12 pt reports 13x12 against the 14x13 default. Template
// keeps AppKit's hover and press tints.
func smallSymbolImage(name, label string) uintptr {
	img := symbolImage(name, label)
	if img == 0 {
		return 0
	}
	smallSymbolOnce.Do(func() { purego.RegisterFunc(&msgSmallSymbolConfig, objcMsgSend) })
	cfg := msgSmallSymbolConfig(class("NSImageSymbolConfiguration"), sel("configurationWithPointSize:weight:scale:"), 12, 0, 2)
	if cfg != 0 {
		if configured := msg(img, "imageWithSymbolConfiguration:", cfg); configured != 0 {
			img = configured
		}
	}
	msg(img, "setTemplate:", 1)
	return img
}

// smallTextButton gives item ti a bordered toolbar button of the small control size as its view:
// AppKit's own text item has no font and no view to apply the 11-point one to.
func (s *toolbarState) smallTextButton(ti, target uintptr, id string, it *ToolbarItem) {
	b := msg(msg(class("NSButton"), "alloc"), "init")
	msg(b, "setButtonType:", pushIn)
	msg(b, "setBezelStyle:", bezelToolbar)
	title := it.Text
	if title == "" {
		title = it.Label
	}
	msg(b, "setTitle:", nsString(title))
	msg(b, "setIdentifier:", nsString(id))
	msg(b, "setTarget:", target)
	msg(b, "setAction:", sel("gowtClick:"))
	msg(b, "setEnabled:", boolArg(!it.Disabled))
	setSmall(b)
	msg(ti, "setView:", b)
	msg(b, "release")
}

// setSmall gives control v the small control size and the system font of that size, which a control
// does not take by itself.
func setSmall(v uintptr) {
	msg(v, "setControlSize:", controlSizeSmall)
	if respondsTo(v, "setFont:") {
		msg(v, "setFont:", uintptr(NSFontSystemFontOfSize(NSFontSystemFontSizeForControlSize(controlSizeSmall)).Id))
	}
}

const (
	// NSTrackingArea options: mouseEntered/Exited, active only while the window is key, so an
	// inactive window shows no hover. The gating covers exited too, so a hover left set when the
	// window resigns key is cleared by the NSWindowDidResignKeyNotification observer below.
	trackingEnteredExited = 0x01 // NSTrackingMouseEnteredAndExited
	trackingActiveInKey   = 0x20 // NSTrackingActiveInKeyWindow
)

var (
	toolbarIconClass uintptr
	// iconButtonSuperDraw is NSButton's drawRect:, which draws the symbol image with its template
	// tint and press dimming; the subclass draws the hover fill around it and forwards.
	iconButtonSuperDraw func(self, sel uintptr, rect NSRect)
	// iconButtonSuperUpdate is NSView's updateTrackingAreas, which the subclass forwards to after
	// re-adding its tracking area: the button is created frameless, so the area of the creation
	// time is empty and AppKit would never deliver entered or exited.
	iconButtonSuperUpdate func(self, sel uintptr)
	iconButtonTracking    func(self, sel uintptr, rect NSRect, options, owner, userInfo uintptr) uintptr
	// iconButtonSuperDealloc is NSView's dealloc; the subclass drops the notification center
	// observer first, so no notification reaches the freed button.
	iconButtonSuperDealloc func(self, sel uintptr)
)

// bareIconButton is the view of a bare single-icon toolbar item. AppKit 26 draws no hover for a
// borderless image item, so the button is a borderless NSButton subclass with a tracking area that
// is active only in the key window: it draws the quiet rounded fill under the pointer and, darker,
// while pressed; the rest state and the sizes stay untouched.
func bareIconButton(target uintptr, id string, img uintptr) uintptr {
	if toolbarIconClass == 0 {
		purego.RegisterFunc(&iconButtonSuperDraw, uintptr(OSClass_getMethodImplementation(int64(class("NSButton")), int64(sel("drawRect:")))))
		purego.RegisterFunc(&iconButtonSuperUpdate, uintptr(OSClass_getMethodImplementation(int64(class("NSView")), int64(sel("updateTrackingAreas")))))
		purego.RegisterFunc(&iconButtonSuperDealloc, uintptr(OSClass_getMethodImplementation(int64(class("NSView")), int64(sel("dealloc")))))
		purego.RegisterFunc(&iconButtonTracking, objcMsgSend)
		draw := purego.NewCallback(func(self, cmd uintptr) {
			if msg(self, "tag") != 0 || msg(self, "isHighlighted")&0xff != 0 {
				if roundedRect == nil {
					purego.RegisterFunc(&roundedRect, objcMsgSend)
				}
				fill := msg(class("NSColor"), "quaternaryLabelColor")
				if msg(self, "isHighlighted")&0xff != 0 && respondsTo(class("NSColor"), "tertiarySystemFillColor") {
					fill = msg(class("NSColor"), "tertiarySystemFillColor")
				}
				r := NewNSViewOverload1(int64(self)).Bounds()
				r.X++
				r.Y++
				r.Width -= 2
				r.Height -= 2
				msg(fill, "setFill")
				msg(roundedRect(class("NSBezierPath"), sel("bezierPathWithRoundedRect:xRadius:yRadius:"), r, 6, 6), "fill")
			}
			iconButtonSuperDraw(self, cmd, NewNSViewOverload1(int64(self)).Bounds())
		})
		entered := purego.NewCallback(func(self, _ uintptr) {
			msg(self, "setTag:", 1)
			msg(self, "setNeedsDisplay:", 1)
		})
		exited := purego.NewCallback(func(self, _ uintptr) {
			msg(self, "setTag:", 0)
			msg(self, "setNeedsDisplay:", 1)
		})
		resign := purego.NewCallback(func(self, _ uintptr) {
			msg(self, "setTag:", 0)
			msg(self, "setNeedsDisplay:", 1)
		})
		dealloc := purego.NewCallback(func(self, cmd uintptr) {
			msg(msg(class("NSNotificationCenter"), "defaultCenter"), "removeObserver:", self)
			iconButtonSuperDealloc(self, cmd)
		})
		update := purego.NewCallback(func(self, cmd uintptr) {
			iconButtonSuperUpdate(self, cmd)
			areas := msg(self, "trackingAreas")
			for i, n := uintptr(0), msg(areas, "count"); i < n; i++ {
				msg(self, "removeTrackingArea:", msg(areas, "objectAtIndex:", 0))
			}
			area := iconButtonTracking(msg(class("NSTrackingArea"), "alloc"), sel("initWithRect:options:owner:userInfo:"),
				NewNSViewOverload1(int64(self)).Bounds(), trackingEnteredExited|trackingActiveInKey, self, 0)
			msg(self, "addTrackingArea:", area)
			msg(area, "release")
		})
		toolbarIconClass = allocClass(class("NSButton"), "GowtToolbarIcon", 0)
		addMethod(toolbarIconClass, sel("drawRect:"), draw, "v@:{CGRect={CGPoint=dd}{CGSize=dd}}")
		addMethod(toolbarIconClass, sel("mouseEntered:"), entered, "v@:@")
		addMethod(toolbarIconClass, sel("mouseExited:"), exited, "v@:@")
		addMethod(toolbarIconClass, sel("gowtResignKey:"), resign, "v@:@")
		addMethod(toolbarIconClass, sel("updateTrackingAreas"), update, "v@:")
		addMethod(toolbarIconClass, sel("dealloc"), dealloc, "v@:")
		registerClass(toolbarIconClass)
	}
	b := msg(msg(toolbarIconClass, "alloc"), "init")
	msg(b, "setTag:", 0) // NSControl starts at -1; the hover state is tag 1
	msg(b, "setButtonType:", pushIn)
	msg(b, "setBordered:", 0)
	msg(b, "setImage:", img)
	msg(b, "setImagePosition:", 1) // NSImageOnly
	msg(b, "setIdentifier:", nsString(id))
	msg(b, "setTarget:", target)
	msg(b, "setAction:", sel("gowtClick:"))
	area := iconButtonTracking(msg(class("NSTrackingArea"), "alloc"), sel("initWithRect:options:owner:userInfo:"),
		NewNSViewOverload1(int64(b)).Bounds(), trackingEnteredExited|trackingActiveInKey, b, 0)
	msg(b, "addTrackingArea:", area)
	msg(area, "release")
	msg(msg(class("NSNotificationCenter"), "defaultCenter"), "addObserver:selector:name:object:",
		b, sel("gowtResignKey:"), nsString("NSWindowDidResignKeyNotification"), 0)
	return b // retained by the toolbar item's setView
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
		if s == nil {
			return
		}
		var id string
		switch {
		case respondsTo(sender, "itemIdentifier"):
			id = goString(msg(sender, "itemIdentifier"))
		case respondsTo(sender, "identifier"): // the button view of a Small item
			id = goString(msg(sender, "identifier"))
		default:
			return
		}
		selected := -1
		if respondsTo(sender, "selectedIndex") {
			selected = int(int64(msg(sender, "selectedIndex")))
		}
		s.clicked(id, selected)
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
	st := &toolbarState{items: append([]ToolbarItem(nil), items...), sidebar: sidebar, collapsed: sidebarHidden[w],
		split: inspectorSplits[w]}
	for i, it := range st.items {
		if it.Center && !it.Space {
			if st.centerIdx == nil {
				st.centerIdx = map[int]bool{}
			}
			st.centerIdx[i] = true
		}
	}
	toolbarDelegates[d] = st
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
		// A bare icon is a custom view; validation leaves the view's own enabled state alone.
		id := itemIDPrefix + strconv.Itoa(item)
		items := msg(tb, "items")
		for i, n := uintptr(0), msg(items, "count"); i < n; i++ {
			if ti := msg(items, "objectAtIndex:", i); goString(msg(ti, "itemIdentifier")) == id {
				if v := msg(ti, "view"); v != 0 && respondsTo(v, "setEnabled:") {
					msg(v, "setEnabled:", boolArg(on))
				}
			}
		}
	}
}

// SetToolbarItemText sets the text that text item number item of the toolbar of the window of view
// shows; a toolbar rebuilt for a sidebar keeps it, since it is stored in the items.
func SetToolbarItemText(view *NSView, item int, text string) {
	w, s := toolbarOf(view)
	if s == nil || item < 0 || item >= len(s.items) {
		return
	}
	s.items[item].Text = text
	tb := msg(w, "toolbar")
	if tb == 0 {
		return
	}
	id := itemIDPrefix + strconv.Itoa(item)
	items := msg(tb, "items")
	for i, n := uintptr(0), msg(items, "count"); i < n; i++ {
		if ti := msg(items, "objectAtIndex:", i); goString(msg(ti, "itemIdentifier")) == id {
			msg(ti, "setTitle:", nsString(text))
			if v := msg(ti, "view"); v != 0 && respondsTo(v, "setTitle:") {
				msg(v, "setTitle:", nsString(text))
			}
		}
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

// toolbarOf is the current toolbar state of the window of view. InstallToolbar swaps the state and the
// NSToolbar (the sidebar calls it again), freeing the old fields, so handles look it up on every call
// and never keep a state or a field.
func toolbarOf(view *NSView) (uintptr, *toolbarState) {
	initAssociated()
	w := msg(uintptr(view.Id), "window")
	if w == 0 {
		return 0, nil
	}
	return w, toolbarDelegates[getAssociated(w, toolbarKey())]
}

// ToolbarSearchField is the first search item of a window's toolbar, for FocusSearch and the QA driver.
type ToolbarSearchField struct{ view *NSView }

// ToolbarSearch returns the handle of the search item of the toolbar of the window of view; it stays valid
// when the toolbar is rebuilt.
func ToolbarSearch(view *NSView) *ToolbarSearchField { return &ToolbarSearchField{view} }

// item is the index of the first search item and its field; the field is 0 until the toolbar has made it.
func (f *ToolbarSearchField) item() (s *toolbarState, i int, field uintptr) {
	_, s = toolbarOf(f.view)
	if s == nil {
		return nil, -1, 0
	}
	for i, it := range s.items {
		if it.Search != nil {
			return s, i, s.searchFields[i]
		}
	}
	return s, -1, 0
}

// Present reports whether the toolbar has a search field now.
func (f *ToolbarSearchField) Present() bool { _, _, sf := f.item(); return sf != 0 }

// ID is the item's ID, "" if it has none.
func (f *ToolbarSearchField) ID() string {
	if s, i, _ := f.item(); i >= 0 {
		return s.items[i].ID
	}
	return ""
}

// Placeholder is the item's title.
func (f *ToolbarSearchField) Placeholder() string {
	if s, i, _ := f.item(); i >= 0 {
		return s.items[i].Label
	}
	return ""
}

// Text is what the field holds.
func (f *ToolbarSearchField) Text() string {
	if _, _, sf := f.item(); sf != 0 {
		return goString(msg(sf, "stringValue"))
	}
	return ""
}

// SetText replaces the text without running the search action, expanding a collapsed field.
func (f *ToolbarSearchField) SetText(t string) {
	if s, i, sf := f.item(); sf != 0 {
		s.showSearch(i, true)
		msg(sf, "setStringValue:", nsString(t))
	}
}

// Focus expands a collapsed field and makes it the first responder at once, without activating the app.
func (f *ToolbarSearchField) Focus() bool {
	s, i, sf := f.item()
	if sf == 0 {
		return false
	}
	w, _ := toolbarOf(f.view)
	s.showSearch(i, true)
	if w == 0 {
		return false
	}
	// The shown field joins the window on the toolbar's next layout; lay it out now.
	if tb := msg(w, "toolbar"); tb != 0 {
		msg(tb, "validateVisibleItems")
	}
	if respondsTo(w, "layoutIfNeeded") {
		msg(w, "layoutIfNeeded")
	}
	return msg(w, "makeFirstResponder:", sf) != 0
}

// Focused reports whether the field or its field editor is the first responder.
func (f *ToolbarSearchField) Focused() bool {
	_, _, sf := f.item()
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
	if s, i, _ := f.item(); i >= 0 && s.items[i].Search != nil {
		s.items[i].Search(f.Text())
	}
}

// Collapse folds an empty field back into its magnifier button, as Esc does.
func (f *ToolbarSearchField) Collapse() {
	if s, i, _ := f.item(); i >= 0 {
		s.showSearch(i, false)
	}
}

// ToolbarButton is a push item or a group segment of the toolbar, as the QA driver lists it. Title is
// the text of a text item, else the label. Bounds is in the coordinates of the view of
// ToolbarButtonsOf, top-left origin; all 0 while AppKit shows no frame for the item.
type ToolbarButton struct {
	ID, Title string
	Enabled   bool
	Bounds    [4]int
}

// ToolbarButtons are the buttons of a window's toolbar, looked up on every call.
type ToolbarButtons struct{ view *NSView }

// ToolbarButtonsOf returns the handle of the buttons of the toolbar of the window of view.
func ToolbarButtonsOf(view *NSView) *ToolbarButtons { return &ToolbarButtons{view} }

// each walks the buttons in order: the items that click, a group as its segments. i is the index of
// the item, j that of the segment, -1 for an item that is no group.
func (b *ToolbarButtons) each(f func(i, j int, it *ToolbarItem) bool) {
	_, s := toolbarOf(b.view)
	if s == nil {
		return
	}
	for i := range s.items {
		it := &s.items[i]
		switch {
		case it.Space || it.Search != nil:
		case len(it.Group) > 0:
			for j := range it.Group {
				if !f(i, j, &it.Group[j]) {
					return
				}
			}
		case it.Click != nil:
			if !f(i, -1, it) {
				return
			}
		}
	}
}

// List returns the buttons in toolbar order.
func (b *ToolbarButtons) List() []ToolbarButton {
	w, s := toolbarOf(b.view)
	if s == nil {
		return nil
	}
	frames := itemFrames(w, b.view)
	var out []ToolbarButton
	b.each(func(i, j int, it *ToolbarItem) bool {
		title := it.Label
		if it.Text != "" {
			title = it.Text
		}
		r := frames[itemIDPrefix+strconv.Itoa(i)]
		if n := len(s.items[i].Group); j >= 0 && n > 0 {
			// AppKit gives no segment frames: the segments are taken as equally wide.
			r = [4]int{r[0] + r[2]*j/n, r[1], r[2] / n, r[3]}
		}
		out = append(out, ToolbarButton{it.ID, title, !it.Disabled, r})
		return true
	})
	return out
}

// itemFrames maps the identifier of each toolbar item on screen to its frame in the coordinates of
// view, top-left origin. An item without a view of its own has no public frame: the AppKit view
// showing it answers item, which is not public API, so an item it misses has no frame.
func itemFrames(w uintptr, view *NSView) map[string][4]int {
	out := map[string][4]int{}
	if w == 0 {
		return out
	}
	content := msg(w, "contentView")
	theme := msg(content, "superview")
	if theme == 0 {
		return out
	}
	flipped := msg(uintptr(view.Id), "isFlipped")&0xff != 0
	height := view.Bounds().Height
	rect := func(v uintptr) [4]int {
		nv := NewNSViewOverload1(int64(v))
		r := nv.ConvertRect_toView_(nv.Bounds(), view)
		if !flipped {
			r.Y = height - r.Y - r.Height
		}
		return [4]int{int(math.Round(r.X)), int(math.Round(r.Y)), int(math.Round(r.Width)), int(math.Round(r.Height))}
	}
	toolbarItem := class("NSToolbarItem")
	var walk func(v uintptr)
	walk = func(v uintptr) {
		if v == content {
			return
		}
		if respondsTo(v, "item") {
			if it := msg(v, "item"); it != 0 && msg(it, "isKindOfClass:", toolbarItem)&0xff != 0 {
				out[goString(msg(it, "itemIdentifier"))] = rect(v)
			}
		}
		subs := msg(v, "subviews")
		for i, n := uintptr(0), msg(subs, "count"); i < n; i++ {
			walk(msg(subs, "objectAtIndex:", i))
		}
	}
	walk(theme)
	if tb := msg(w, "toolbar"); tb != 0 {
		items := msg(tb, "items")
		for i, n := uintptr(0), msg(items, "count"); i < n; i++ {
			ti := msg(items, "objectAtIndex:", i)
			id := goString(msg(ti, "itemIdentifier"))
			if _, ok := out[id]; ok {
				continue
			}
			if v := msg(ti, "view"); v != 0 && msg(v, "window") == w {
				out[id] = rect(v)
			}
		}
	}
	return out
}

// Click runs the handler of button n of List, as a click on it does.
func (b *ToolbarButtons) Click(n int) bool {
	ok := false
	b.each(func(_, _ int, it *ToolbarItem) bool {
		if n == 0 {
			if it.Click != nil && !it.Disabled {
				it.Click()
				ok = true
			}
			return false
		}
		n--
		return true
	})
	return ok
}

// toolbarIconButton is the bare icon button of the toolbar item with the user-visible ID of the
// window of view; 0 without the item or before the toolbar has given it a view.
func toolbarIconButton(view *NSView, id string) uintptr {
	w, s := toolbarOf(view)
	if s == nil || id == "" {
		return 0
	}
	for i, it := range s.items {
		if it.ID != id {
			continue
		}
		tb := msg(w, "toolbar")
		if tb == 0 {
			return 0
		}
		items := msg(tb, "items")
		for n, c := uintptr(0), msg(items, "count"); n < c; n++ {
			ti := msg(items, "objectAtIndex:", n)
			if goString(msg(ti, "itemIdentifier")) != itemIDPrefix+strconv.Itoa(i) {
				continue
			}
			if v := msg(ti, "view"); v != 0 && respondsTo(v, "tag") {
				return v
			}
		}
	}
	return 0
}

// ToolbarIconHovered reports the hover state of the bare icon button of the toolbar item with the
// user-visible ID of the window of view, as mouseEntered: sets it; false without the item.
func ToolbarIconHovered(view *NSView, id string) bool {
	v := toolbarIconButton(view, id)
	return v != 0 && msg(v, "tag") != 0
}

// ToolbarIconSendHover sends the button of the toolbar item with the user-visible ID the
// mouseEntered: or mouseExited: message, as the tracking area delivers it in the key window.
// The hover GUI checks drive the mechanism this way: moving the real pointer is off limits there.
func ToolbarIconSendHover(view *NSView, id string, entered bool) bool {
	v := toolbarIconButton(view, id)
	if v == 0 {
		return false
	}
	if entered {
		msg(v, "mouseEntered:", 0)
	} else {
		msg(v, "mouseExited:", 0)
	}
	return true
}

// ToolbarPostResignKey posts NSWindowDidResignKeyNotification with the window as the object,
// as AppKit does: observers such as NSRemoteView assert an NSWindow and raise on nil.
func ToolbarPostResignKey(view *NSView) {
	msg(msg(class("NSNotificationCenter"), "defaultCenter"), "postNotificationName:object:",
		nsString("NSWindowDidResignKeyNotification"), msg(uintptr(view.Id), "window"))
}
