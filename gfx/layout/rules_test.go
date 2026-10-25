package layout

import "testing"

func fixed(v float64) Size { return Fixed(v) }
func fill() Size           { return Fill }

type rect struct{ x, y, w, h float64 }

func frame(n *Node) rect { return rect{n.RX, n.RY, n.RW, n.RH} }

func check(t *testing.T, name string, n *Node, want rect) {
	t.Helper()
	if got := frame(n); got != want {
		t.Errorf("%s: got %v, want %v", name, got, want)
	}
}

func TestMinMax(t *testing.T) {
	tests := []struct {
		name string
		node *Node
		want rect
	}{
		{"hug width raised to min", &Node{MinW: 80, Kids: []*Node{box(20, 10)}}, rect{0, 0, 80, 10}},
		{"hug width cut to max", &Node{MaxW: 15, Kids: []*Node{box(20, 10)}}, rect{0, 0, 15, 10}},
		{"fixed width cut to max", &Node{W: fixed(100), MaxW: 60, H: fixed(5)}, rect{0, 0, 60, 5}},
		{"fixed height raised to min", &Node{W: fixed(10), H: fixed(5), MinH: 9}, rect{0, 0, 10, 9}},
		{"hug height raised to min", &Node{Dir: Column, MinH: 50, Kids: []*Node{box(20, 10)}}, rect{0, 0, 20, 50}},
		{"hug height cut to max", &Node{Dir: Column, MaxH: 4, Kids: []*Node{box(20, 10)}}, rect{0, 0, 20, 4}},
		{"min wins over max", &Node{W: fixed(10), H: fixed(1), MinW: 30, MaxW: 20}, rect{0, 0, 30, 1}},
	}
	for _, tt := range tests {
		tt.node.Layout()
		check(t, tt.name, tt.node, tt.want)
	}

	// a Fill child cut by Max does not pass the rest of its share to the sibling
	a := &Node{W: fill(), H: fixed(5), MaxW: 20}
	b := &Node{W: fill(), H: fixed(5)}
	row := &Node{W: fixed(100), Kids: []*Node{a, b}}
	row.Layout()
	if a.RW != 20 || b.RW != 50 {
		t.Errorf("fill with max: %v %v", a.RW, b.RW)
	}

	// a min height moves the aligned children
	c := box(10, 10)
	r := &Node{MinH: 30, Align: Center, Kids: []*Node{c}}
	r.Layout()
	if c.RY != 10 {
		t.Errorf("centered in a min height: %v", c.RY)
	}
}

func TestJustifyAlignColumn(t *testing.T) {
	tests := []struct {
		name string
		j, a Align
		want []rect
	}{
		{"start", Start, Start, []rect{{0, 0, 10, 10}, {0, 10, 20, 10}}},
		{"center", Center, Center, []rect{{45, 40, 10, 10}, {40, 50, 20, 10}}},
		{"end", End, End, []rect{{90, 80, 10, 10}, {80, 90, 20, 10}}},
		{"space between", SpaceBetween, Start, []rect{{0, 0, 10, 10}, {0, 90, 20, 10}}},
	}
	for _, tt := range tests {
		a, b := box(10, 10), box(20, 10)
		root := &Node{Dir: Column, W: fixed(100), H: fixed(100), Justify: tt.j, Align: tt.a, Kids: []*Node{a, b}}
		root.Layout()
		check(t, tt.name+" a", a, tt.want[0])
		check(t, tt.name+" b", b, tt.want[1])
	}
}

func TestWrapRow(t *testing.T) {
	mk := func() (*Node, []*Node) {
		k := []*Node{box(40, 10), box(40, 20), box(40, 10), box(20, 5)}
		return &Node{W: fixed(100), Gap: 5, LineGap: 7, Wrap: true, Kids: k}, k
	}
	root, k := mk()
	root.Layout()
	// line 1: 40+5+40 = 85 (a third would need 130), line 2: 40+5+20 = 65
	check(t, "k0", k[0], rect{0, 0, 40, 10})
	check(t, "k1", k[1], rect{45, 0, 40, 20})
	check(t, "k2", k[2], rect{0, 27, 40, 10})
	check(t, "k3", k[3], rect{45, 27, 20, 5})
	if root.RH != 37 { // 20 + 7 + 10, the hug height
		t.Errorf("hug height %v", root.RH)
	}

	// padding, justify and align work inside a line
	root, k = mk()
	root.Pad = [4]float64{3, 3, 3, 3}
	root.Align, root.Justify = End, End
	root.H = fixed(60)
	root.Layout()
	check(t, "padded k0", k[0], rect{12, 13, 40, 10})
	check(t, "padded k2", k[2], rect{32, 30, 40, 10})

	// a hug width is one line
	root = &Node{Wrap: true, Gap: 5, Kids: []*Node{box(40, 10), box(40, 10), box(40, 10)}}
	root.Layout()
	if root.RW != 130 || root.RH != 10 {
		t.Errorf("hug wrap row %v x %v", root.RW, root.RH)
	}
	// the same row with a max width wraps
	root.MaxW = 90
	root.Layout()
	if root.RW != 90 || root.RH != 20 {
		t.Errorf("hug wrap row with MaxW %v x %v", root.RW, root.RH)
	}
}

func TestWrapRowFillGrows(t *testing.T) {
	a := &Node{W: fill(), H: fixed(10), LeafW: func() float64 { return 30 }}
	b := box(50, 10)
	c := &Node{W: fill(), H: fixed(10), LeafW: func() float64 { return 40 }}
	root := &Node{W: fixed(100), Gap: 10, Wrap: true, Kids: []*Node{a, b, c}}
	root.Layout()
	// line 1: a(30) + 10 + b(50) = 90, free 10, a grows to 40; c goes to line 2 and grows to 100
	check(t, "a", a, rect{0, 0, 40, 10})
	check(t, "b", b, rect{50, 0, 50, 10})
	check(t, "c", c, rect{0, 10, 100, 10})
}

func TestWrapSpaceBetweenPerLine(t *testing.T) {
	k := []*Node{box(30, 10), box(30, 10), box(30, 10), box(30, 10)}
	root := &Node{W: fixed(100), Wrap: true, Justify: SpaceBetween, Kids: k}
	root.Layout()
	// three fit in 100, the fourth goes alone to line 2
	if k[2].RX != 70 || k[3].RX != 0 || k[3].RY != 10 {
		t.Errorf("k2 %v k3 %v", frame(k[2]), frame(k[3]))
	}
}

func TestWrapColumn(t *testing.T) {
	k := []*Node{box(10, 40), box(20, 40), box(15, 40), box(5, 10)}
	root := &Node{Dir: Column, H: fixed(100), Gap: 5, LineGap: 8, Wrap: true, Kids: k}
	root.Layout()
	// column 1: 40+5+40 = 85; column 2: 40+5+10 = 55; widths 20 and 15
	check(t, "k0", k[0], rect{0, 0, 10, 40})
	check(t, "k1", k[1], rect{0, 45, 20, 40})
	check(t, "k2", k[2], rect{28, 0, 15, 40})
	check(t, "k3", k[3], rect{28, 45, 5, 10})
	if root.RW != 43 || root.RH != 100 { // 20 + 8 + 15, the hug width
		t.Errorf("root %v x %v", root.RW, root.RH)
	}
}

func TestWrapColumnWithoutHeightIsOneColumn(t *testing.T) {
	k := []*Node{box(10, 40), box(20, 40)}
	root := &Node{Dir: Column, Wrap: true, Kids: k}
	root.Layout()
	if k[1].RX != 0 || k[1].RY != 40 || root.RH != 80 {
		t.Errorf("k1 %v root %v", frame(k[1]), frame(root))
	}
}

func TestWrapTextShrinksToLine(t *testing.T) {
	var at float64
	text := &Node{W: fill(), LeafW: func() float64 { return 200 }, LeafH: func(w float64) float64 { at = w; return 12 * float64(int(200/w)+1) }}
	root := &Node{W: fixed(100), Wrap: true, Kids: []*Node{box(10, 5), text}}
	root.Layout()
	// the text (200) does not fit next to the box, takes line 2 alone, shrinks to 100 and wraps there
	if at != 100 {
		t.Errorf("text wrapped at %v", at)
	}
	check(t, "text", text, rect{0, 5, 100, 36})
}

func TestAbsoluteChildInFlow(t *testing.T) {
	a, b := box(20, 10), box(30, 10)
	badge := box(8, 8)
	badge.Absolute, badge.X, badge.Y = true, 50, -4
	root := &Node{Gap: 5, Pad: [4]float64{2, 2, 2, 2}, Kids: []*Node{a, badge, b}}
	root.Layout()
	// the badge takes no gap, no width and no height
	check(t, "root", root, rect{0, 0, 2 + 20 + 5 + 30 + 2, 14})
	check(t, "b", b, rect{27, 2, 30, 10})
	check(t, "badge", badge, rect{50, -4, 8, 8})

	// an absolute Fill child covers the parent's box
	veil := &Node{Absolute: true, W: fill(), H: fill()}
	col := &Node{Dir: Column, W: fixed(40), H: fixed(30), Kids: []*Node{box(10, 10), veil}}
	col.Layout()
	check(t, "veil", veil, rect{0, 0, 40, 30})

	// only absolute children: the node hugs its padding
	only := &Node{Pad: [4]float64{1, 2, 3, 4}, Kids: []*Node{{Absolute: true, W: fixed(9), H: fixed(9)}}}
	only.Layout()
	check(t, "only absolute", only, rect{0, 0, 6, 4})
}

func TestLayoutTwiceSeesChanges(t *testing.T) {
	k := box(10, 10)
	root := &Node{Kids: []*Node{k}}
	root.Layout()
	k.W = fixed(30)
	root.Layout()
	if root.RW != 30 {
		t.Errorf("second layout width %v", root.RW)
	}
}

func TestHugTextWithMaxWidthWraps(t *testing.T) {
	var at float64
	text := &Node{MaxW: 100, LeafW: func() float64 { return 250 }, LeafH: func(w float64) float64 { at = w; return 10 * float64(int(250/w)+1) }}
	text.Layout()
	if at != 100 {
		t.Errorf("wrapped at %v", at)
	}
	check(t, "text", text, rect{0, 0, 100, 30})
}
