package layout

import "testing"

func box(w, h float64) *Node { return &Node{W: Size{Fixed, w}, H: Size{Fixed, h}} }

func TestColumnHugAndFill(t *testing.T) {
	a, b := box(50, 10), box(20, 30)
	fill := &Node{W: Size{Fill, 0}, H: Size{Fixed, 5}}
	root := &Node{Dir: Column, Gap: 4, Pad: [4]float64{2, 3, 2, 3}, Kids: []*Node{a, b, fill}}
	root.Layout()
	// width hugs the widest child plus padding, height is the sum of children, gaps and padding
	if root.RW != 56 || root.RH != 10+30+5+8+4 {
		t.Fatalf("root %v x %v", root.RW, root.RH)
	}
	if b.RX != 3 || b.RY != 2+10+4 {
		t.Errorf("b at %v,%v", b.RX, b.RY)
	}
	if fill.RW != 50 {
		t.Errorf("fill width %v, want the inner width 50", fill.RW)
	}
}

func TestRowFillShareJustifyAlign(t *testing.T) {
	a := box(20, 10)
	f1 := &Node{W: Size{Fill, 0}, H: Size{Fixed, 6}}
	f2 := &Node{W: Size{Fill, 0}, H: Size{Fixed, 6}}
	root := &Node{W: Size{Fixed, 100}, H: Size{Fixed, 20}, Gap: 10, Align: Center, Kids: []*Node{a, f1, f2}}
	root.Layout()
	if f1.RW != 30 || f2.RW != 30 { // (100 - 20 - 2*10) / 2
		t.Errorf("fill widths %v %v", f1.RW, f2.RW)
	}
	if a.RY != 5 || f1.RY != 7 {
		t.Errorf("cross offsets %v %v", a.RY, f1.RY)
	}
	if f2.RX != 20+10+30+10 {
		t.Errorf("f2 at %v", f2.RX)
	}

	x, y := box(10, 10), box(10, 10)
	sb := &Node{W: Size{Fixed, 100}, H: Size{Fixed, 10}, Justify: SpaceBetween, Kids: []*Node{x, y}}
	sb.Layout()
	if y.RX != 90 {
		t.Errorf("space between puts the last child at %v", y.RX)
	}
	c := box(10, 10)
	ce := &Node{W: Size{Fixed, 100}, H: Size{Fixed, 10}, Justify: Center, Kids: []*Node{c}}
	ce.Layout()
	if c.RX != 45 {
		t.Errorf("centered at %v", c.RX)
	}
}

func TestLeafWrapsAtAssignedWidth(t *testing.T) {
	var at float64
	text := &Node{W: Size{Fill, 0}, LeafW: func() float64 { return 200 }, LeafH: func(w float64) float64 { at = w; return 12 * float64(int(200/w)+1) }}
	root := &Node{Dir: Column, W: Size{Fixed, 100}, Kids: []*Node{text, box(10, 5)}}
	root.Layout()
	if at != 100 || text.RH != 36 {
		t.Errorf("text laid out at %v, height %v", at, text.RH)
	}
	if root.RH != 41 {
		t.Errorf("root height %v: the wrapped text must push the sibling", root.RH)
	}
}

func TestAbsHug(t *testing.T) {
	k := box(10, 10)
	k.X, k.Y = 30, 40
	root := &Node{Dir: Abs, Kids: []*Node{k}}
	root.Layout()
	if root.RW != 40 || root.RH != 50 || k.RX != 30 {
		t.Errorf("abs root %v x %v, kid at %v", root.RW, root.RH, k.RX)
	}
}
