package look

import (
	"testing"

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
