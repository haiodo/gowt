package look

import (
	"fmt"
	"testing"

	"github.com/haiodo/gowt/svg"
	"github.com/haiodo/gowt/swt"
)

func TestBarForeground(t *testing.T) {
	black, white := swt.RGB{Red: 0, Green: 0, Blue: 0}, swt.RGB{Red: 255, Green: 255, Blue: 255}
	dark := swt.RGB{Red: 30, Green: 31, Blue: 34}
	if got := barForeground(black, white); got != black {
		t.Errorf("readable system colour replaced: %v", got)
	}
	if got := barForeground(black, dark); luma(got) <= 128 {
		t.Errorf("black text kept on a dark bar: %v", got)
	}
	if got := barForeground(white, white); luma(got) > 128 {
		t.Errorf("white text kept on a light bar: %v", got)
	}
	if got := *blend(black, white, 40); got != (swt.RGB{Red: 102, Green: 102, Blue: 102}) {
		t.Errorf("blend 40%%: %v", got)
	}
}

func TestBarSymbolsRender(t *testing.T) {
	for name, shape := range barSymbols {
		data := fmt.Sprintf(barSVG, shape)
		for _, zoom := range []int32{100, 200} {
			img := svg.NewImageDataProvider([]byte(data), 16, 16, "#000000").GetImageData(zoom)
			if img == nil {
				t.Errorf("%s: does not render at zoom %d", name, zoom)
				continue
			}
			inked := false
			for y := int32(0); y < img.Height && !inked; y++ {
				for x := int32(0); x < img.Width; x++ {
					if img.GetAlpha(x, y) > 0 {
						inked = true
						break
					}
				}
			}
			if !inked {
				t.Errorf("%s: blank at zoom %d", name, zoom)
			}
		}
	}
}

// The bar splits the items into the zones of the three panes: the sidebar row (leading and
// SidebarEnd trailing), the content bar (Nav leading, Center, the trailing rest) and the inspector
// strip; without the pane the pinned items fall back to the trailing zone.
func TestBarZones(t *testing.T) {
	items := []ToolbarItem{
		{Title: "open", Sidebar: true},
		{Title: "collapse", SidebarEnd: true},
		{Title: "back", Nav: true},
		{Title: "zoom", Center: true},
		{Title: "add"},
		{Title: "info", Inspector: true},
		{Space: true, Inspector: true},
		{Title: "toggle", Inspector: true},
	}
	over, sideEnd, lead, centered, trail, insp, spaced := barZones(items, true, true)
	if len(over) != 1 || over[0].Title != "open" {
		t.Errorf("sidebar leading %v", over)
	}
	if len(sideEnd) != 1 || sideEnd[0].Title != "collapse" {
		t.Errorf("sidebar trailing %v", sideEnd)
	}
	if len(lead) != 1 || lead[0].Title != "back" {
		t.Errorf("content leading %v", lead)
	}
	if len(centered) != 1 || centered[0].Title != "zoom" {
		t.Errorf("content center %v", centered)
	}
	if len(trail) != 1 || trail[0].Title != "add" || spaced {
		t.Errorf("content trailing %v spaced %v", trail, spaced)
	}
	if len(insp) != 3 || !insp[1].Space {
		t.Errorf("inspector strip %v", insp)
	}
	// Without the panes the pinned items are ordinary trailing ones.
	_, sideEnd, _, centered, trail, insp, _ = barZones(items, false, false)
	if len(sideEnd) != 0 || len(centered) != 1 || len(insp) != 0 || len(trail) != 6 {
		t.Errorf("no panes: sideEnd %v centered %v trail %v insp %v", sideEnd, centered, trail, insp)
	}
}
