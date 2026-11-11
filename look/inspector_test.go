package look

import "testing"

func TestInspectorFrames(t *testing.T) {
	// macOS 26: edge to edge, the glass from the top of the window, the sash just inside its edge.
	bg, ib, ct, sh := inspectorFrames(0, 0, 900, 600, 260, 0, 52, 0)
	if bg != [4]int32{640, 0, 260, 600} || ib != [4]int32{640, 52, 260, 548} {
		t.Errorf("inspector background %v, inspector %v", bg, ib)
	}
	if ct != [4]int32{0, 52, 640, 548} {
		t.Errorf("content %v", ct)
	}
	if sh != [4]int32{640, 52, sashGrip, 548} {
		t.Errorf("sash inside the edge %v", sh)
	}
	// Right of a sidebar: the span starts at the sidebar's content x.
	_, _, sct, _ := sidebarFrames(0, 0, 900, 600, 220, 7, 52, 0)
	_, ib, ct, _ = inspectorFrames(sct[0], 0, sct[2], 600, 260, 0, 52, 0)
	if ct != [4]int32{227, 52, 413, 548} || ib[0]+ib[2] != 900 {
		t.Errorf("between sidebar and inspector: content %v, inspector %v", ct, ib)
	}
	// Windows and Linux: the sash fills a gap of its own between content and inspector.
	_, ib, ct, sh = inspectorFrames(0, 0, 900, 600, 260, 0, 0, 4)
	if ib[0] != 640 || sh != [4]int32{636, 0, 4, 600} || ct[2] != 636 {
		t.Errorf("divider gap: inspector %v sash %v content %v", ib, sh, ct)
	}
	// An inset inspector floats: the background is inset on three sides.
	bg, ib, ct, _ = inspectorFrames(0, 0, 900, 600, 260, 7, 52, 0)
	if bg != [4]int32{633, 7, 260, 586} || ib != [4]int32{633, 52, 260, 541} || ct[2] != 633 {
		t.Errorf("inset 7: background %v, inspector %v, content %v", bg, ib, ct)
	}
	bg, ib, ct, _ = inspectorFrames(0, 0, 100, 10, 260, 7, 52, 0)
	if bg[3] != 0 || ib[3] != 0 || ct[2] != 0 || ct[3] != 0 {
		t.Errorf("a window smaller than the inspector gave negative sizes: %v %v %v", bg, ib, ct)
	}
}

func TestInspectorHiddenFrames(t *testing.T) {
	for _, div := range []int32{0, 4} {
		bg, ib, ct, sh := inspectorFrames(10, 0, 900, 600, 0, 0, 52, div)
		if ct != [4]int32{10, 52, 900, 548} {
			t.Errorf("div %d: hidden inspector left content %v, want the whole span", div, ct)
		}
		if bg[2] != 0 || ib[2] != 0 || sh[0]+sh[2] > 910 {
			t.Errorf("div %d: hidden inspector background %v, inspector %v, sash %v", div, bg, ib, sh)
		}
	}
}

// inspectorWidthAt undoes inspectorFrames: the sash placed for a width gives that width back.
func TestInspectorWidthAt(t *testing.T) {
	for _, c := range []struct{ x, in, div int32 }{{0, 0, 0}, {227, 0, 0}, {0, 7, 0}, {0, 0, 4}} {
		for _, width := range []int32{160, 261, 400} {
			_, _, _, sh := inspectorFrames(c.x, 0, 1000-c.x, 600, width, c.in, 52, c.div)
			if got := inspectorWidthAt(sh[0], 1000, c.in, c.div); got != width {
				t.Errorf("x %d in %d div %d: width %d, sash at %d gives %d", c.x, c.in, c.div, width, sh[0], got)
			}
		}
	}
}

// The inspector asks only for its 160-point minimum: strip items that do not fit overflow into
// the More menu, so a requested width below the strip's preferred size stays settable.
func TestInspectorMinWidth(t *testing.T) {
	p := &pane{kind: inspectorPane}
	if got := p.minWidth(); got != minInspectorWidth() {
		t.Errorf("inspector minWidth %d, want the %d minimum", got, minInspectorWidth())
	}
}

// room is what a pane takes from the content, nothing for a missing or hidden pane.
func TestPaneRoom(t *testing.T) {
	var none *pane
	if none.room() != 0 || none.visible() || none.widthOrZero() != 0 {
		t.Error("a missing pane takes room or reports a state")
	}
	p := &pane{kind: inspectorPane, width: 260}
	if got := p.room(); got != inspectorInset()+260+sidebarDivider() {
		t.Errorf("inspector room %d", got)
	}
	p.hidden = true
	if p.room() != 0 || p.shown() != 0 {
		t.Errorf("hidden inspector: room %d, shown %d", p.room(), p.shown())
	}
}

func TestBottomFrames(t *testing.T) {
	// macOS 26: no gap, the sash just inside the panel's top edge.
	ct, pb, sh := bottomFrames(227, 52, 413, 548, 160, 0)
	if pb != [4]int32{227, 440, 413, 160} || ct != [4]int32{227, 52, 413, 388} || sh != [4]int32{227, 440, 413, sashGrip} {
		t.Errorf("div 0: content %v, panel %v, sash %v", ct, pb, sh)
	}
	// Windows and Linux: the sash fills a gap of its own above the panel.
	ct, pb, sh = bottomFrames(0, 30, 900, 570, 160, 4)
	if pb != [4]int32{0, 440, 900, 160} || ct != [4]int32{0, 30, 900, 406} || sh != [4]int32{0, 436, 900, 4} {
		t.Errorf("div 4: content %v, panel %v, sash %v", ct, pb, sh)
	}
	// Hidden: the content keeps the column, the panel is 0 high at its foot.
	for _, div := range []int32{0, 4} {
		ct, pb, _ = bottomFrames(0, 30, 900, 570, 0, div)
		if ct != [4]int32{0, 30, 900, 570} || pb[3] != 0 {
			t.Errorf("div %d hidden: content %v, panel %v", div, ct, pb)
		}
	}
	ct, _, _ = bottomFrames(0, 30, 900, 50, 160, 4)
	if ct[3] != 0 {
		t.Errorf("a column lower than the panel gave content %v", ct)
	}
}

// bottomHeightAt undoes bottomFrames: the sash placed for a height gives that height back.
func TestBottomHeightAt(t *testing.T) {
	for _, div := range []int32{0, 4} {
		for _, height := range []int32{60, 161, 300} {
			_, _, sh := bottomFrames(0, 52, 900, 548, height, div)
			if got := bottomHeightAt(sh[1], 600, div); got != height {
				t.Errorf("div %d: height %d, sash at %d gives %d", div, height, sh[1], got)
			}
		}
	}
}

func TestPanesOverlayFrames(t *testing.T) {
	l := &paneLayout{overlay: true}
	l.side = &pane{l: l, kind: sidebarPane, width: 220}
	l.insp = &pane{l: l, kind: inspectorPane, width: 260}
	client := [4]int32{10, 20, 900, 600}
	for _, hidden := range []bool{false, true} {
		l.side.hidden, l.insp.hidden = hidden, hidden
		if ct := l.overlayFrames(client); ct != client {
			t.Fatalf("hidden %v: content %v, want the whole client %v", hidden, ct, client)
		}
		if got := l.requiredWidth(900); got != 900 {
			t.Errorf("hidden %v: width %d", hidden, got)
		}
	}
	l.side.hidden, l.insp.hidden = false, false
	// The side panes run the full client height; the bottom panel floats at the foot of the
	// central column between their inner edges, over the full-height content.
	in := sidebarInset() // in overlay both side panes float with the sidebar's glass inset
	_, sb, col, _ := sidebarFrames(client[0], client[1], client[2], client[3], 220, in, 52, sidebarDivider())
	_, ib, col, _ := inspectorFrames(col[0], client[1], col[2], client[3], 260, in, 52, sidebarDivider())
	if sb[1]+sb[3] != client[1]+client[3]-in {
		t.Errorf("sidebar %v does not run to the client foot %d but for the inset %d", sb, client[1]+client[3], in)
	}
	if ib[1]+ib[3] != client[1]+client[3]-in {
		t.Errorf("inspector %v does not run to the client foot %d but for the inset %d", ib, client[1]+client[3], in)
	}
	l.bottom = &pane{kind: bottomPane, width: 160}
	ct := l.overlayFrames(client)
	_, pb, _ := bottomFrames(col[0], col[1], col[2], col[3], l.bottom.shown(), sidebarDivider())
	if ct != client {
		t.Errorf("content %v, want the whole client %v", ct, client)
	}
	if pb[0] != sb[0]+sb[2] || pb[0]+pb[2] != ib[0] {
		t.Errorf("bottom %v, want between the sidebar edge %d and the inspector %v", pb, sb[0]+sb[2], ib)
	}
	if pb[3] != 160 || pb[1]+pb[3] != client[1]+client[3] {
		t.Errorf("bottom %v, want 160 high at the client foot", pb)
	}
	l.overlay = false
	if got := l.requiredWidth(900); got != 900+l.side.room()+l.insp.room() {
		t.Errorf("legacy width %d", got)
	}
}

func TestPanesOverlayDragLimits(t *testing.T) {
	l := &paneLayout{overlay: true}
	l.side = &pane{l: l, kind: sidebarPane, width: 220}
	l.insp = &pane{l: l, kind: inspectorPane, width: 260}
	for _, p := range []*pane{l.side, l.insp} {
		overlay := l.paneLimit(p, 900)
		if got := dragWidth(1000, 160, overlay); got != overlay {
			t.Errorf("drag %d, limit %d", got, overlay)
		}
		l.overlay = false
		want := p.width + 900 - l.side.room() - l.insp.room() - minContentWidth()
		if legacy := l.paneLimit(p, 900); legacy != want {
			t.Errorf("legacy limit %d, overlay %d", legacy, overlay)
		}
		l.overlay = true
	}
}

// The full-height switch drops the pane's floating inset: the sidebar loses it always, the
// inspector in a panes overlay window (elsewhere it is flush already).
func TestPaneInsetFullHeight(t *testing.T) {
	l := &paneLayout{overlay: true}
	l.side = &pane{l: l, kind: sidebarPane}
	l.insp = &pane{l: l, kind: inspectorPane}
	if got := l.insp.inset(); got != sidebarInset() {
		t.Errorf("floating overlay inspector inset %d, want the sidebar inset %d", got, sidebarInset())
	}
	l.inspectorFullHeight = true
	if got := l.insp.inset(); got != inspectorInset() {
		t.Errorf("full-height inspector inset %d, want the flush %d", got, inspectorInset())
	}
	l.sidebarFullHeight = true
	if got := l.side.inset(); got != 0 {
		t.Errorf("full-height sidebar inset %d, want 0", got)
	}
}

// In a panes overlay the floating panes' children stay a glass inset off the rounded bottom
// corners of the glass; the legacy layout keeps its frames.
func TestPaneFramesFloatingCorners(t *testing.T) {
	overlay := &paneLayout{overlay: true}
	overlay.side = &pane{l: overlay, kind: sidebarPane, width: 220}
	overlay.insp = &pane{l: overlay, kind: inspectorPane, width: 260}
	_, sb, _, _ := overlay.side.sidebarFrames(10, 20, 900, 600, 52)
	if want := int32(20 + 600 - 2*sidebarInset()); sb[1]+sb[3] != want {
		t.Errorf("overlay sidebar children run to %d, want %d: off the rounded corners", sb[1]+sb[3], want)
	}
	_, ib, _, _ := overlay.insp.inspectorFrames(237, 20, 656, 600, 52)
	if want := int32(20 + 600 - 2*sidebarInset()); ib[1]+ib[3] != want {
		t.Errorf("overlay inspector children run to %d, want %d: off the rounded corners", ib[1]+ib[3], want)
	}
	legacy := &paneLayout{}
	legacy.side = &pane{l: legacy, kind: sidebarPane, width: 220}
	_, sb, _, _ = legacy.side.sidebarFrames(10, 20, 900, 600, 52)
	if want := int32(20 + 600 - sidebarInset()); sb[1]+sb[3] != want {
		t.Errorf("legacy sidebar children run to %d, want %d", sb[1]+sb[3], want)
	}
}
