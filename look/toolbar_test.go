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
