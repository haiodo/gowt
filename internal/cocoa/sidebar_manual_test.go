//go:build darwin

package cocoa

import (
	"strings"
	"testing"

	"github.com/ebitengine/purego"
)

func TestGroupedFillRGB(t *testing.T) {
	onMain(func() {
		r, g, b := GroupedFillRGB()
		if r < 0 || r > 255 || g < 0 || g > 255 || b < 0 || b > 255 {
			t.Errorf("rgb %d %d %d", r, g, b)
		}
	})
}

func TestToolbarClicked(t *testing.T) {
	var got []string
	s := &toolbarState{items: []ToolbarItem{
		{Label: "plain", Click: func() { got = append(got, "plain") }},
		{Label: "nav", Group: []ToolbarItem{
			{Label: "back", Click: func() { got = append(got, "back") }},
			{Label: "forward", Click: func() { got = append(got, "forward") }},
		}},
		{Label: "find", Search: func(string) {}},
	}}
	s.clicked(itemIDPrefix+"0", -1)
	s.clicked(itemIDPrefix+"1", 1)
	s.clicked(itemIDPrefix+"1", -1)
	s.clicked(itemIDPrefix+"1", 2)
	s.clicked(itemIDPrefix+"2", -1)
	s.clicked(sidebarSpacerID, -1)
	if len(got) != 2 || got[0] != "plain" || got[1] != "forward" {
		t.Errorf("clicks %v", got)
	}
}

func TestToolbarEnabled(t *testing.T) {
	onMain(func() {
		s := &toolbarState{items: []ToolbarItem{
			{Label: "plain", Disabled: true},
			{Label: "nav", Group: []ToolbarItem{
				{Label: "back", Symbol: "chevron.backward", Disabled: true},
				{Label: "forward", Symbol: "chevron.forward"},
			}},
		}}
		plain := s.item(0, itemIDPrefix+"0")
		if s.enabled(plain) || msg(plain, "isEnabled")&0xff != 0 {
			t.Error("a disabled push item is enabled")
		}
		subs := msg(s.item(0, itemIDPrefix+"1"), "subitems")
		if n := msg(subs, "count"); n != 2 {
			t.Fatalf("group has %d subitems", n)
		}
		back, forward := msg(subs, "objectAtIndex:", 0), msg(subs, "objectAtIndex:", 1)
		if s.enabled(back) || !s.enabled(forward) || msg(back, "isEnabled")&0xff != 0 {
			t.Error("segment state not applied")
		}
		s.items[1].Group[0].Disabled = false
		if !s.enabled(back) {
			t.Error("validation does not follow the item state")
		}
	})
}

func TestToolbarSidebarSpacer(t *testing.T) {
	onMain(func() {
		s := &toolbarState{items: []ToolbarItem{{Label: "a"}, {Space: true}}}
		if n := msg(s.ids(), "count"); n != 2 {
			t.Errorf("without a sidebar: %d ids", n)
		}
		s.sidebar = true
		ids := s.ids()
		if n := msg(ids, "count"); n != 3 || goString(msg(ids, "objectAtIndex:", 0)) != sidebarSpacerID {
			t.Errorf("with a sidebar: %d ids, first %q", n, goString(msg(ids, "objectAtIndex:", 0)))
		}
	})
}

func TestToolbarSpacerItem(t *testing.T) {
	onMain(func() {
		s := &toolbarState{sidebar: true}
		ti := s.item(0, sidebarSpacerID)
		if ti == 0 || s.width == 0 || msg(ti, "view") != s.spacer || msg(ti, "isNavigational")&0xff == 0 {
			t.Fatalf("spacer item %x view %x width %x", ti, s.spacer, s.width)
		}
	})
}

// setStyle: fits the columns into the table width whatever the column autoresizing style: in the
// fusion-desktop sidebar (gowt-issues 4) a 44-point column became 13 points wide.
func TestSourceListKeepsColumnWidths(t *testing.T) {
	onMain(func() {
		sv := NewNSScrollViewOverload1(int64(msg(msg(class("NSScrollView"), "alloc"), "init")))
		defer sv.Release()
		sv.SetFrame(NSRect{Width: 260, Height: 300})
		tv := NewNSTableViewOverload1(int64(msg(msg(msg(class("NSTableView"), "alloc"), "init"), "autorelease")))
		tv.SetColumnAutoresizingStyle(int64(OSNSTableViewNoColumnAutoresizing))
		tv.SetHeaderView(nil)
		msg(uintptr(tv.Id), "setStyle:", uintptr(OSNSTableViewStylePlain))
		tv.SetIntercellSpacing(NSSize{Width: 1, Height: 1})
		want := []float64{185, 43, 29} // with the spacing exactly as wide as the clip view, as the Resize listener sets them
		for _, w := range want {
			c := NewNSTableColumnOverload1(int64(msg(msg(msg(class("NSTableColumn"), "alloc"), "initWithIdentifier:", nsString("c")), "autorelease")))
			c.SetWidth(w)
			tv.AddTableColumn(c)
		}
		msg(uintptr(sv.Id), "setDocumentView:", uintptr(tv.Id))
		tv.Tile()
		SetSourceList(&tv.NSView)
		cols := tv.TableColumns()
		for i, w := range want {
			if got := NewNSTableColumnOverload1(cols.ObjectAtIndex(int64(i)).Id).Width(); got != w {
				t.Errorf("column %d: %.1f wide, want %.1f", i, got, w)
			}
		}
	})
}

func TestSourceListHooks(t *testing.T) {
	onMain(func() {
		tv := msg(msg(class("NSTableView"), "alloc"), "init")
		plain := msg(msg(class("NSTableView"), "alloc"), "init")
		SetSourceList(NewNSViewOverload1(int64(tv)))
		red := msg(class("NSColor"), "redColor")
		msg(tv, "setBackgroundColor:", red)
		msg(plain, "setBackgroundColor:", red)
		if msg(tv, "backgroundColor") == red {
			t.Error("a source list took an opaque background")
		}
		if msg(plain, "backgroundColor") != red {
			t.Error("the hook changed a table that is no source list")
		}
		rv := msg(tv, "tableView:rowViewForRow:", tv, 0)
		if rv == 0 || msg(rv, "isEmphasized")&0xff != 0 {
			t.Fatalf("source-list row view %x", rv)
		}
		msg(rv, "setSelected:", 1) // through the override; needsDisplay is not kept for a view outside a window
		if msg(rv, "isSelected")&0xff == 0 {
			t.Error("setSelected: did not reach NSTableRowView")
		}
		if msg(tv, "selectionHighlightStyle") != ^uintptr(0) {
			t.Error("AppKit still draws the selection")
		}
		if rv := msg(plain, "tableView:rowViewForRow:", plain, 0); rv != 0 {
			t.Errorf("plain table got row view %x", rv)
		}
	})
}

// SWT reports a selection change from the delegate method; SetSourceList used to trigger it by
// re-applying the selected rows, a spurious Selection event for a sidebar list selected before look.SetSidebar.
func TestSourceListKeepsSelectionQuiet(t *testing.T) {
	onMain(func() {
		changes := 0
		rows := purego.NewCallback(func(self, _, table uintptr) int { return 3 })
		changed := purego.NewCallback(func(self, _, note uintptr) { changes++ })
		cell := purego.NewCallback(func(self, _, table, col uintptr, row int) uintptr {
			return msg(msg(msg(class("NSTableCellView"), "alloc"), "init"), "autorelease")
		})
		// Like an SWT table: view-based, its own data source and delegate.
		cls := allocClass(class("NSTableView"), "GowtTestSelfDelegateTable", 0)
		addMethod(cls, sel("numberOfRowsInTableView:"), rows, "q@:@")
		addMethod(cls, sel("tableView:viewForTableColumn:row:"), cell, "@@:@@q")
		addMethod(cls, sel("tableViewSelectionDidChange:"), changed, "v@:@")
		registerClass(cls)
		tv := msg(msg(cls, "alloc"), "init")
		msg(tv, "addTableColumn:", msg(msg(class("NSTableColumn"), "alloc"), "initWithIdentifier:", nsString("c")))
		msg(tv, "setDataSource:", tv)
		msg(tv, "setDelegate:", tv)
		msg(tv, "reloadData")
		msg(tv, "selectRowIndexes:byExtendingSelection:", msg(class("NSIndexSet"), "indexSetWithIndex:", 1), 0)
		if changes != 1 {
			t.Fatalf("selecting a row gave %d change notifications, want 1", changes)
		}
		SetSourceList(NewNSViewOverload1(int64(tv)))
		if changes != 1 {
			t.Errorf("SetSourceList reported %d selection changes", changes-1)
		}
		if row := msg(tv, "selectedRow"); row != 1 || msg(tv, "delegate") != tv {
			t.Errorf("selected row %d, delegate %x", int(row), msg(tv, "delegate"))
		}
	})
}

func TestDescribeViewChain(t *testing.T) {
	onMain(func() {
		host := msg(msg(class("NSScrollView"), "alloc"), "init")
		tv := msg(msg(class("NSTableView"), "alloc"), "init")
		msg(host, "setDocumentView:", tv)
		out := DescribeViewChain(NewNSViewOverload1(int64(tv)))
		if !strings.Contains(out, "NSTableView") || !strings.Contains(out, "NSScrollView") || !strings.Contains(out, "drawsBackground=") {
			t.Errorf("chain:\n%s", out)
		}
	})
}

func TestToolbarSearch(t *testing.T) {
	onMain(func() {
		registerToolbarClass()
		d := msg(msg(toolbarClass, "alloc"), "init")
		defer msg(d, "release")
		var got []string
		toolbarDelegates[d] = &toolbarState{items: []ToolbarItem{{Label: "find", Search: func(s string) { got = append(got, s) }}}}
		defer delete(toolbarDelegates, d)
		f := msg(toolbarDelegates[d].item(d, itemIDPrefix+"0"), "searchField")
		msg(f, "setStringValue:", nsString("web"))
		msg(f, "sendAction:to:", msg(f, "action"), msg(f, "target"))
		if len(got) != 1 || got[0] != "web" {
			t.Errorf("search got %v", got)
		}
		// The window's field editor is SWT's and drops the text typed into a field of no SWT widget.
		cell, fe := msg(f, "cell"), sel("fieldEditorForView:")
		if msg(msg(cell, "class"), "instanceMethodForSelector:", fe) == msg(class("NSSearchFieldCell"), "instanceMethodForSelector:", fe) {
			t.Error("the search cell takes the window's field editor")
		}
		if msg(cell, "fieldEditorForView:", f) != 0 {
			t.Error("a field outside a window gets a field editor")
		}
		w := msg(msg(class("NSObject"), "alloc"), "init")
		defer msg(w, "release")
		ed := searchEditor(w)
		if ed == 0 || ed != searchEditor(w) || msg(ed, "isFieldEditor")&0xff == 0 ||
			goString(msg(msg(ed, "className"), "description")) != "NSTextView" {
			t.Errorf("search field editor %x", ed)
		}
	})
}

// Sidebar items go between the sidebar spacer and the end spacer, before everything else; without
// a sidebar they keep their place.
func TestToolbarSidebarOrder(t *testing.T) {
	items := []ToolbarItem{{Label: "back", Nav: true}, {Label: "open", Sidebar: true}, {Space: true, Sidebar: true}, {Label: "add"}}
	order := func(s *toolbarState) string {
		var ids []string
		onMain(func() {
			arr := s.ids()
			for i, n := uintptr(0), msg(arr, "count"); i < n; i++ {
				ids = append(ids, goString(msg(arr, "objectAtIndex:", i)))
			}
		})
		return strings.Join(ids, " ")
	}
	want := "gowt.sidebar gowt.item.1 gowt.sidebar.end gowt.item.0 NSToolbarFlexibleSpaceItem gowt.item.3"
	if got := order(&toolbarState{items: items, sidebar: true}); got != want {
		t.Errorf("with sidebar: %s, want %s", got, want)
	}
	want = "gowt.item.0 gowt.item.1 NSToolbarFlexibleSpaceItem gowt.item.3"
	if got := order(&toolbarState{items: items}); got != want {
		t.Errorf("without sidebar: %s, want %s", got, want)
	}
	// A hidden sidebar drops the spacers; the Sidebar items stay first, after the traffic lights.
	want = "gowt.item.1 gowt.item.0 NSToolbarFlexibleSpaceItem gowt.item.3"
	if got := order(&toolbarState{items: items, sidebar: true, collapsed: true}); got != want {
		t.Errorf("sidebar hidden: %s, want %s", got, want)
	}
}

// Inspector items go last, after the tracking separator, once the window has an inspector split view;
// without one they keep their place.
func TestToolbarInspectorOrder(t *testing.T) {
	items := []ToolbarItem{{Label: "info", Inspector: true}, {Label: "back", Nav: true}, {Space: true, Inspector: true}, {Label: "add"}}
	order := func(s *toolbarState) string {
		var ids []string
		onMain(func() {
			arr := s.ids()
			for i, n := uintptr(0), msg(arr, "count"); i < n; i++ {
				ids = append(ids, goString(msg(arr, "objectAtIndex:", i)))
			}
		})
		return strings.Join(ids, " ")
	}
	want := "gowt.item.1 gowt.item.3 gowt.inspector gowt.item.0 NSToolbarFlexibleSpaceItem"
	if got := order(&toolbarState{items: items, split: 1}); got != want {
		t.Errorf("with inspector: %s, want %s", got, want)
	}
	want = "gowt.item.0 gowt.item.1 NSToolbarFlexibleSpaceItem gowt.item.3"
	if got := order(&toolbarState{items: items}); got != want {
		t.Errorf("without inspector: %s, want %s", got, want)
	}
}

func TestToolbarInspectorSeparator(t *testing.T) {
	onMain(func() {
		sv := msg(msg(class("NSSplitView"), "alloc"), "init")
		defer msg(sv, "release")
		for range 2 {
			msg(sv, "addSubview:", msg(msg(msg(class("NSView"), "alloc"), "init"), "autorelease"))
		}
		s := &toolbarState{items: []ToolbarItem{{Label: "info", Inspector: true}}, split: sv}
		ti := s.item(0, inspectorID)
		if ti == 0 || msg(ti, "isKindOfClass:", class("NSTrackingSeparatorToolbarItem"))&0xff == 0 || msg(ti, "splitView") != sv {
			t.Fatalf("separator item %x", ti)
		}
		if (&toolbarState{}).item(0, inspectorID) != 0 {
			t.Error("a separator without a split view")
		}
	})
}

func TestInspectorDivider(t *testing.T) {
	onMain(func() {
		sv := msg(msg(class("NSSplitView"), "alloc"), "init")
		defer msg(sv, "release")
		msg(sv, "setVertical:", 1)
		for range 2 {
			msg(sv, "addSubview:", msg(msg(msg(class("NSView"), "alloc"), "init"), "autorelease"))
		}
		msgRectOnly(sv, sel("setFrame:"), NSRect{Width: 900})
		// The view is in no window, so its window is 0.
		inspectorSplits[0] = sv
		defer delete(inspectorSplits, 0)
		v := NewNSViewOverload1(int64(msg(msg(class("NSView"), "alloc"), "init")))
		defer v.Release()
		subs := msg(sv, "subviews")
		first, second := NewNSViewOverload1(int64(msg(subs, "objectAtIndex:", 0))), msg(subs, "objectAtIndex:", 1)
		SetInspectorDivider(v, 640, false)
		if w := first.Frame().Width; w != 640 {
			t.Errorf("divider at 640: first pane %.1f wide", w)
		}
		SetInspectorDivider(v, 900, true)
		if msg(second, "isHidden")&0xff == 0 || msg(sv, "isSubviewCollapsed:", second)&0xff == 0 {
			t.Error("a hidden inspector does not collapse the second pane")
		}
		SetInspectorDivider(v, 560, false)
		if w := first.Frame().Width; msg(second, "isHidden")&0xff != 0 || w != 560 {
			t.Errorf("shown again at 560: hidden %d, first pane %.1f wide", msg(second, "isHidden")&0xff, w)
		}
	})
}

// A text item shows its text as a bordered title; a Small item gets a small button view whose click
// runs the item.
func TestToolbarTextAndSmall(t *testing.T) {
	onMain(func() {
		registerToolbarClass()
		d := msg(msg(toolbarClass, "alloc"), "init")
		defer msg(d, "release")
		clicks := 0
		s := &toolbarState{items: []ToolbarItem{
			{Label: "Zoom", Text: "100%", Click: func() { clicks++ }},
			{Label: "Fit", Text: "Fit", Small: true, Click: func() { clicks++ }},
			{Label: "Add", Symbol: "plus", Small: true, Disabled: true},
		}}
		toolbarDelegates[d] = s
		defer delete(toolbarDelegates, d)
		text := s.item(d, itemIDPrefix+"0")
		if goString(msg(text, "title")) != "100%" || msg(text, "isBordered")&0xff == 0 || msg(text, "view") != 0 {
			t.Errorf("text item: title %q, bordered %d", goString(msg(text, "title")), msg(text, "isBordered")&0xff)
		}
		small := s.item(d, itemIDPrefix+"1")
		b := msg(small, "view")
		if b == 0 || b != s.buttons[1] || msg(b, "controlSize") != controlSizeSmall || goString(msg(b, "title")) != "Fit" {
			t.Fatalf("small item view %x, size %d, title %q", b, msg(b, "controlSize"), goString(msg(b, "title")))
		}
		msg(b, "performClick:", 0)
		if clicks != 1 {
			t.Errorf("small button clicks %d", clicks)
		}
		add := msg(s.item(d, itemIDPrefix+"2"), "view")
		if msg(add, "image") == 0 || msg(add, "isEnabled")&0xff != 0 {
			t.Errorf("small symbol item: image %x, enabled %d", msg(add, "image"), msg(add, "isEnabled")&0xff)
		}
	})
}
