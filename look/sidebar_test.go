package look

import (
	"testing"

	"github.com/haiodo/gowt"
)

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
	}{
		{300, 120, 600, 300},
		{120, 120, 600, 120},
		{119, 120, 600, 120}, // the drag stops, the sidebar stays
		{-20, 120, 600, 120},
		{900, 120, 600, 600},
		{300, 120, 50, 120},  // a narrow window still keeps the minimum
		{170, 180, 600, 180}, // Sidebar items that need 180 points
	} {
		if w := dragWidth(c.want, c.least, c.most); w != c.width {
			t.Errorf("dragWidth(%d, %d, %d) = %d, want %d", c.want, c.least, c.most, w, c.width)
		}
	}
}

// The sidebar narrows to 88 points: its toolbar items overflow instead of holding it wider.
func TestMinSidebarWidth(t *testing.T) {
	if got, want := minSidebarWidth(), int32(gowt.Scaled(88)); got != want {
		t.Errorf("minSidebarWidth() = %d, want %d", got, want)
	}
}

func TestSidebarFullHeightFrames(t *testing.T) {
	l := &paneLayout{sidebarFullHeight: true}
	p := &pane{l: l, kind: sidebarPane, width: 220}
	bg, sb, ct, sh := p.sidebarFrames(10, 20, 900, 600, 52)
	if bg != [4]int32{10, 20, 220, 600} || sb != bg {
		t.Fatalf("full-height background %v, sidebar %v", bg, sb)
	}
	if ct != [4]int32{230 + sidebarDivider(), 72, 680 - sidebarDivider(), 548} || sh[1] != 20 || sh[3] != 600 {
		t.Fatalf("full-height content %v, sash %v", ct, sh)
	}
	if p.floats() || p.inset() != 0 {
		t.Fatal("full-height sidebar floats")
	}
	l.overlay = true
	if p.floats() || p.inset() != 0 {
		t.Fatal("overlay made full-height sidebar float")
	}
	l.sidebarFullHeight = false
	gotBG, gotSB, gotCT, gotSH := p.sidebarFrames(10, 20, 900, 600, 52)
	wantBG, wantSB, wantCT, wantSH := sidebarFrames(10, 20, 900, 600, 220, sidebarInset(), 52, sidebarDivider())
	if gotBG != wantBG || gotSB != wantSB || gotCT != wantCT || gotSH != wantSH {
		t.Fatalf("disabled mode changed default frames: %v %v %v %v", gotBG, gotSB, gotCT, gotSH)
	}
}

func TestSidebarFullHeightHiddenFrames(t *testing.T) {
	l := &paneLayout{sidebarFullHeight: true}
	p := &pane{l: l, kind: sidebarPane, width: 220, hidden: true}
	bg, sb, ct, _ := p.sidebarFrames(10, 20, 900, 600, 52)
	if bg[2] != 0 || sb[2] != 0 || ct != [4]int32{10, 72, 900, 548} {
		t.Fatalf("hidden full-height frames: %v %v %v", bg, sb, ct)
	}
	p.hidden = false
	_, sb, ct, _ = p.sidebarFrames(10, 20, 100, 10, 52)
	if sb[3] != 10 || ct[2] != 0 || ct[3] != 0 {
		t.Fatalf("small full-height window: sidebar %v content %v", sb, ct)
	}
}
