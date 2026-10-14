package look

import "testing"

func TestSidebarFrames(t *testing.T) {
	bg, sb, ct := sidebarFrames(0, 0, 900, 600, 220, 7, 52)
	if bg != [4]int32{7, 7, 220, 586} || sb != [4]int32{7, 52, 220, 541} {
		t.Errorf("sidebar background %v, sidebar %v", bg, sb)
	}
	if ct != [4]int32{227, 52, 673, 548} {
		t.Errorf("content %v", ct)
	}
	bg, sb, ct = sidebarFrames(0, 0, 100, 10, 220, 7, 52)
	if bg[3] != 0 || sb[3] != 0 || ct[2] != 0 || ct[3] != 0 {
		t.Errorf("a window smaller than the sidebar gave negative sizes: %v %v %v", bg, sb, ct)
	}
	if bg, sb, _ := sidebarFrames(0, 0, 900, 600, 220, 0, 0); bg != sb {
		t.Errorf("without inset and title strip the background is the sidebar: %v %v", bg, sb)
	}
}
