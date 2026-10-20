package look

import "testing"

func TestSidebarFrames(t *testing.T) {
	bg, sb, ct, sh := sidebarFrames(0, 0, 900, 600, 220, 7, 52, 0)
	if bg != [4]int32{7, 7, 220, 586} || sb != [4]int32{7, 52, 220, 541} {
		t.Errorf("sidebar background %v, sidebar %v", bg, sb)
	}
	if ct != [4]int32{227, 52, 673, 548} {
		t.Errorf("content %v", ct)
	}
	if sh != [4]int32{224, 52, sashGrip, 548} {
		t.Errorf("sash over the edge %v", sh)
	}
	bg, sb, ct, _ = sidebarFrames(0, 0, 100, 10, 220, 7, 52, 0)
	if bg[3] != 0 || sb[3] != 0 || ct[2] != 0 || ct[3] != 0 {
		t.Errorf("a window smaller than the sidebar gave negative sizes: %v %v %v", bg, sb, ct)
	}
	if bg, sb, _, _ := sidebarFrames(0, 0, 900, 600, 220, 0, 0, 0); bg != sb {
		t.Errorf("without inset and title strip the background is the sidebar: %v %v", bg, sb)
	}
	// Windows and Linux: the sash fills a gap of its own between sidebar and content.
	_, sb, ct, sh = sidebarFrames(0, 0, 900, 600, 220, 0, 0, 4)
	if sb[0]+sb[2] != 220 || sh != [4]int32{220, 0, 4, 600} || ct[0] != 224 || ct[2] != 676 {
		t.Errorf("divider gap: sidebar %v sash %v content %v", sb, sh, ct)
	}
}

func TestSidebarHiddenFrames(t *testing.T) {
	for _, div := range []int32{0, 4} {
		bg, sb, ct, sh := sidebarFrames(10, 0, 900, 600, 0, 7, 52, div)
		if ct != [4]int32{10, 52, 900, 548} {
			t.Errorf("div %d: hidden sidebar left content %v, want the whole width", div, ct)
		}
		if bg[2] != 0 || sb[2] != 0 || sh[0] < 10 {
			t.Errorf("div %d: hidden sidebar background %v, sidebar %v, sash %v", div, bg, sb, sh)
		}
	}
}

// sidebarWidthAt undoes sidebarFrames: the sash placed for a width gives that width back.
func TestSidebarWidthAt(t *testing.T) {
	for _, c := range []struct{ in, div int32 }{{7, 0}, {0, 0}, {0, 4}} {
		for _, width := range []int32{120, 221, 400} {
			_, _, _, sh := sidebarFrames(0, 0, 1000, 600, width, c.in, 52, c.div)
			if got := sidebarWidthAt(sh[0], c.in, c.div); got != width {
				t.Errorf("in %d div %d: width %d, sash at %d gives %d", c.in, c.div, width, sh[0], got)
			}
		}
	}
}

func TestDragWidth(t *testing.T) {
	for _, c := range []struct {
		want, least, most, width int32
		shown                    bool
	}{
		{300, 120, 600, 300, true},
		{120, 120, 600, 120, true},
		{119, 120, 600, 0, false},
		{-20, 120, 600, 0, false},
		{900, 120, 600, 600, true},
		{300, 120, 50, 120, true}, // a narrow window still keeps the minimum
		{170, 180, 600, 0, false}, // Sidebar items that need 180 points
		{180, 180, 600, 180, true},
	} {
		if w, ok := dragWidth(c.want, c.least, c.most); w != c.width || ok != c.shown {
			t.Errorf("dragWidth(%d, %d, %d) = %d, %v; want %d, %v", c.want, c.least, c.most, w, ok, c.width, c.shown)
		}
	}
}
