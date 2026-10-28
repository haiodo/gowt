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
