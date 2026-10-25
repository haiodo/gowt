// Package layout is a flexbox-like auto-layout for editors and design tools: rows and columns with gap,
// padding, alignment, justification, wrapping, min and max sizes, absolute children, and fixed, hug and
// fill sizes. It knows nothing of drawing or of any file format: leaves that depend on content (text)
// report their size through two callbacks. Only scene.go imports gfx (to apply frames to a gfx.Scene);
// the rest uses the standard library, so it can be copied into another module without that file.
//
// Usage: build a tree of Node, call Layout on the root, read RX, RY, RW, RH of every node (relative to
// the parent's top left corner), or hand the tree to Apply to move gfx.Scene nodes.
//
// Rules:
//
//   - Dir is Row (children left to right, the default), Column, or Abs (all children at their own
//     X, Y). In a Row or a Column a child with Absolute set leaves the flow: it takes no part in gap,
//     fill shares or hug sizes, and sits at X, Y from the parent's top left corner (padding ignored).
//   - A size is Fixed (a number), Hug (the content plus padding) or Fill (a share of the free main
//     axis space of the parent, the whole cross axis of the parent). Fill children of one parent share
//     the free space equally. An absolute child fills the parent's whole box.
//   - MinW, MaxW, MinH, MaxH clamp the resolved size of any kind; zero Max means no limit, and Min wins
//     over Max. A Fill child clamped by Max does not give the rest of its share to its siblings.
//   - Widths are decided top down, from the intrinsic widths of the content; heights bottom up, from
//     the widths. A Hug height, or a Fill height under a parent of unknown height, is the content's.
//   - Gap goes between children. Padding is top, right, bottom, left.
//   - Justify (main axis) is Start, Center, End or SpaceBetween; Align (cross axis) is Start, Center or End.
//   - Wrap breaks the children of a Row or a Column into lines when they do not fit the main axis; LineGap
//     goes between lines, Gap between children of a line, Justify and Align work inside a line, and lines
//     start at the cross start. A Row wraps at its width, so give it a Fixed or Fill width, or a MaxW;
//     a Hug width is one line. A Column wraps at its height, which must be known: Fixed, or Fill under
//     a parent of known height. A Fill child of a wrapping node starts from its content size and takes an equal
//     share of its line's free space, or gives back an equal share of the overflow.
//   - Leaves (no children) with LeafW and LeafH set take their size from them: LeafW is the natural
//     width, LeafH(w) the height at width w (text wrapping). A Hug text is as wide as LeafW, a Fill
//     text, or a Hug one with MaxW, is wrapped by LeafH at the width it gets.
//
// Not supported: per-child align overrides, percentages, Fill sizes with a fallback value, content
// alignment of wrapped lines.
package layout

import "math"

// Dir is how a node places its children.
type Dir uint8

// The directions of a Node: children left to right, top to bottom, or each at its own X, Y.
const (
	Row Dir = iota
	Column
	Abs
)

// Kind is how a size is decided.
type Kind uint8

// The kinds of Size. Use Fixed, Hug and Fill to make one.
const (
	KindHug Kind = iota
	KindFixed
	KindFill
)

// Size is a width or a height: Fixed(v), Hug or Fill. The zero value is Hug.
type Size struct {
	Kind Kind
	V    float64 // the number of a Fixed size
}

// Fixed is a size of v points.
func Fixed(v float64) Size { return Size{KindFixed, v} }

var (
	// Hug is the size of the content plus padding.
	Hug = Size{}
	// Fill is an equal share of the free space of the parent's main axis, the whole cross axis.
	Fill = Size{Kind: KindFill}
)

// Insets are the four paddings of a Node: top, right, bottom, left.
type Insets [4]float64

// All is the same padding v on every side.
func All(v float64) Insets { return Insets{v, v, v, v} }

// XY is x on the left and the right, y on the top and the bottom.
func XY(x, y float64) Insets { return Insets{y, x, y, x} }

// Align places children on an axis. SpaceBetween is for Justify only.
type Align uint8

// The alignments. SpaceBetween is for Justify only.
const (
	Start Align = iota
	Center
	End
	SpaceBetween
)

// Node is one box. Set the inputs, call Layout on the root, read RX, RY, RW, RH.
type Node struct {
	W, H Size
	// Min and max sizes; a zero Max is no limit.
	MinW, MaxW, MinH, MaxH float64

	X, Y     float64 // position in an Abs parent, or of an Absolute child
	Absolute bool    // leaves the flow of a Row or a Column parent

	Dir     Dir
	Gap     float64
	Pad     Insets
	Align   Align
	Justify Align

	Wrap    bool    // break children into lines (Row, Column)
	LineGap float64 // between lines of a wrapping node

	Kids []*Node

	// LeafW and LeafH make a leaf depend on its content. LeafH is also called with the final width
	// when the height is already known, so a text can wrap itself at the width it got.
	LeafW func() float64
	LeafH func(w float64) float64

	// Tag is for the caller, for example the app's own node; Apply passes it back.
	Tag any

	// Results, relative to the parent's top left corner.
	RX, RY, RW, RH float64

	iw   float64
	iwOK bool
}

func clamp(v, lo, hi float64) float64 {
	if hi > 0 && v > hi {
		v = hi
	}
	if v < lo {
		v = lo
	}
	return v
}

func (n *Node) clampW(w float64) float64 { return clamp(w, n.MinW, n.MaxW) }
func (n *Node) clampH(h float64) float64 { return clamp(h, n.MinH, n.MaxH) }

// flow returns the children that take part in the flow of a Row or a Column.
func (n *Node) flow() []*Node {
	for _, k := range n.Kids {
		if k.Absolute {
			out := make([]*Node, 0, len(n.Kids))
			for _, k := range n.Kids {
				if !k.Absolute {
					out = append(out, k)
				}
			}
			return out
		}
	}
	return n.Kids
}

func gapsOf(gap float64, n int) float64 { return gap * float64(max(n-1, 0)) }

func (n *Node) forget() {
	n.iwOK = false
	for _, k := range n.Kids {
		k.forget()
	}
}

// Layout sizes n from its own W and H (Hug and Fill count as Hug at the root) and places everything
// below. It can be called again after the tree or the inputs changed.
func (n *Node) Layout() {
	n.forget()
	w := n.intrinsicW()
	if n.W.Kind == KindFixed {
		w = n.clampW(n.W.V)
	}
	h := -1.0
	if n.H.Kind == KindFixed {
		h = n.H.V
	}
	n.layout(w, h)
	n.RX, n.RY = 0, 0
}

func (n *Node) intrinsicW() float64 {
	if n.W.Kind == KindFixed {
		return n.clampW(n.W.V)
	}
	if n.iwOK {
		return n.iw
	}
	var w float64
	kids := n.Kids
	if n.Dir != Abs {
		kids = n.flow()
	}
	switch {
	case len(n.Kids) == 0:
		if n.LeafW != nil {
			w = n.LeafW()
		}
	case n.Dir == Abs:
		for _, k := range kids {
			w = math.Max(w, k.X+k.intrinsicW())
		}
	case n.Dir == Column && n.Wrap && n.H.Kind == KindFixed:
		w = n.wrappedColumnsW(kids)
	case n.Dir == Row:
		w = n.Pad[1] + n.Pad[3] + gapsOf(n.Gap, len(kids))
		for _, k := range kids {
			w += k.intrinsicW()
		}
	default:
		for _, k := range kids {
			w = math.Max(w, k.intrinsicW())
		}
		w += n.Pad[1] + n.Pad[3]
	}
	w = n.clampW(w)
	n.iw, n.iwOK = w, true
	return w
}

// wrappedColumnsW is the width of a wrapping Column of fixed height: its columns side by side.
// It lays the kids out at their natural width; the real layout redoes it.
func (n *Node) wrappedColumnsW(kids []*Node) float64 {
	avail := n.clampH(n.H.V) - n.Pad[0] - n.Pad[2]
	w, colW, colH, cols := 0.0, 0.0, 0.0, 0
	for i, k := range kids {
		kh := -1.0
		if k.H.Kind == KindFixed {
			kh = k.H.V
		}
		k.layout(k.intrinsicW(), kh)
		if i > 0 && colH+n.Gap+k.RH > avail+1e-9 {
			w, colW, colH, cols = w+colW, 0, 0, cols+1
		} else if i > 0 {
			colH += n.Gap
		}
		colW, colH = math.Max(colW, k.RW), colH+k.RH
	}
	if len(kids) > 0 {
		w, cols = w+colW, cols+1
	}
	return w + n.LineGap*float64(max(cols-1, 0)) + n.Pad[1] + n.Pad[3]
}

func (n *Node) kidW(avail float64) float64 {
	switch n.W.Kind {
	case KindFixed:
		return n.clampW(n.W.V)
	case KindFill:
		return n.clampW(avail)
	}
	return n.intrinsicW()
}

// layout sizes n to width w and height h, h < 0 meaning as high as the content.
func (n *Node) layout(w, h float64) {
	w = n.clampW(w)
	if h >= 0 {
		h = n.clampH(h)
	}
	n.RW = w
	switch {
	case len(n.Kids) == 0:
		n.RH = math.Max(h, 0)
		if h < 0 && n.LeafH != nil {
			n.RH = n.clampH(n.LeafH(w))
		} else if n.LeafH != nil {
			n.LeafH(w) // the leaf still wraps at w
		}
	case n.Dir == Abs:
		n.layoutAbs(w, h)
	default:
		n.layoutFlow(w, h)
		n.layoutAbsolute()
	}
}

func (n *Node) layoutAbs(w, h float64) {
	maxB := 0.0
	for _, k := range n.Kids {
		kh := -1.0
		switch k.H.Kind {
		case KindFixed:
			kh = k.H.V
		case KindFill:
			kh = h
		}
		k.layout(k.kidW(w), kh)
		k.RX, k.RY = k.X, k.Y
		maxB = math.Max(maxB, k.RY+k.RH)
	}
	n.RH = h
	if h < 0 {
		n.RH = n.clampH(maxB)
	}
}

// layoutAbsolute places the Absolute children of a Row or a Column, once n has its size.
func (n *Node) layoutAbsolute() {
	for _, k := range n.Kids {
		if !k.Absolute {
			continue
		}
		kh := -1.0
		switch k.H.Kind {
		case KindFixed:
			kh = k.H.V
		case KindFill:
			kh = n.RH
		}
		k.layout(k.kidW(n.RW), kh)
		k.RX, k.RY = k.X, k.Y
	}
}

func (n *Node) layoutFlow(w, h float64) {
	kids := n.flow()
	if n.Wrap {
		n.layoutWrap(kids, w, h)
		return
	}
	pt, pr, pb, pl := n.Pad[0], n.Pad[1], n.Pad[2], n.Pad[3]
	innerW := w - pl - pr
	gaps := gapsOf(n.Gap, len(kids))
	innerH := h - pt - pb
	if h < 0 {
		innerH = -1
	}
	if n.Dir == Row {
		fixed, nfill := 0.0, 0
		for _, k := range kids {
			if k.W.Kind == KindFill {
				nfill++
			} else {
				fixed += k.kidW(0)
			}
		}
		share := 0.0
		if nfill > 0 {
			share = math.Max((innerW-fixed-gaps)/float64(nfill), 0)
		}
		contentH := 0.0
		for _, k := range kids {
			kh := -1.0
			switch k.H.Kind {
			case KindFixed:
				kh = k.H.V
			case KindFill:
				kh = innerH
			}
			k.layout(k.kidW(share), kh)
			contentH = math.Max(contentH, k.RH)
		}
		if h < 0 {
			h, innerH = n.hugH(contentH)
			for _, k := range kids {
				if k.H.Kind == KindFill {
					k.layout(k.RW, innerH)
				}
			}
		}
		n.RH = h
		n.place(kids, true, pl, pt, innerW, innerH, gaps)
		return
	}
	sum, nfill := 0.0, 0
	for _, k := range kids {
		if k.H.Kind == KindFill && innerH >= 0 {
			nfill++
			continue
		}
		kh := -1.0
		if k.H.Kind == KindFixed {
			kh = k.H.V
		}
		k.layout(k.kidW(innerW), kh)
		sum += k.RH
	}
	if nfill > 0 {
		share := math.Max((innerH-sum-gaps)/float64(nfill), 0)
		for _, k := range kids {
			if k.H.Kind == KindFill {
				k.layout(k.kidW(innerW), share)
				sum += k.RH
			}
		}
	}
	if h < 0 {
		h, innerH = n.hugH(sum + gaps)
	}
	n.RH = h
	n.place(kids, false, pl, pt, innerW, innerH, gaps)
}

// hugH is the height of a node that hugs content of height content, and the room inside its padding.
func (n *Node) hugH(content float64) (h, inner float64) {
	h = n.clampH(content + n.Pad[0] + n.Pad[2])
	inner = content
	if h != content+n.Pad[0]+n.Pad[2] {
		inner = h - n.Pad[0] - n.Pad[2]
	}
	return h, inner
}

type line struct {
	kids        []*Node
	main, cross float64
}

// layoutWrap lays out a wrapping Row or Column: kids get their natural main size and are cut into
// lines, Fill kids grow into the free space of their line, then each line is placed like a flow.
func (n *Node) layoutWrap(kids []*Node, w, h float64) {
	pt, pr, pb, pl := n.Pad[0], n.Pad[1], n.Pad[2], n.Pad[3]
	horiz := n.Dir == Row
	innerW, innerH := w-pl-pr, h-pt-pb
	if h < 0 {
		innerH = -1
	}
	mainAvail := innerW
	if !horiz {
		mainAvail = innerH
		if innerH < 0 {
			mainAvail = math.Inf(1)
		}
	}
	mainOf := func(k *Node) float64 {
		if horiz {
			return k.RW
		}
		return k.RH
	}
	crossOf := func(k *Node) float64 {
		if horiz {
			return k.RH
		}
		return k.RW
	}
	growsMain := func(k *Node) bool {
		if horiz {
			return k.W.Kind == KindFill
		}
		return k.H.Kind == KindFill
	}
	fixedH := func(k *Node) float64 {
		if k.H.Kind == KindFixed {
			return k.H.V
		}
		return -1
	}

	var lines []*line
	cur := &line{}
	for _, k := range kids {
		k.layout(k.intrinsicW(), fixedH(k))
		m := mainOf(k)
		if len(cur.kids) > 0 && cur.main+n.Gap+m > mainAvail+1e-9 {
			lines = append(lines, cur)
			cur = &line{}
		}
		if len(cur.kids) > 0 {
			cur.main += n.Gap
		}
		cur.main += m
		cur.kids = append(cur.kids, k)
	}
	if len(cur.kids) > 0 {
		lines = append(lines, cur)
	}

	maxMain, total := 0.0, 0.0
	for i, ln := range lines {
		nfill := 0
		for _, k := range ln.kids {
			if growsMain(k) {
				nfill++
			}
		}
		if free := mainAvail - ln.main; nfill > 0 && free != 0 && !math.IsInf(free, 1) {
			ln.main = gapsOf(n.Gap, len(ln.kids))
			for _, k := range ln.kids {
				share := free / float64(nfill)
				switch {
				case growsMain(k) && horiz:
					k.layout(math.Max(k.RW+share, 0), fixedH(k))
				case growsMain(k):
					k.layout(k.RW, math.Max(k.RH+share, 0))
				}
				ln.main += mainOf(k)
			}
		}
		for _, k := range ln.kids {
			ln.cross = math.Max(ln.cross, crossOf(k))
		}
		for _, k := range ln.kids {
			switch {
			case horiz && k.H.Kind == KindFill:
				k.layout(k.RW, ln.cross)
			case !horiz && k.W.Kind == KindFill:
				k.layout(ln.cross, fixedH(k))
			}
		}
		maxMain = math.Max(maxMain, ln.main)
		total += ln.cross
		if i > 0 {
			total += n.LineGap
		}
	}

	if h < 0 {
		content := total
		if !horiz {
			content = maxMain
		}
		h, innerH = n.hugH(content)
	}
	if !horiz {
		mainAvail = innerH
	}
	n.RH = h
	off := 0.0
	for _, ln := range lines {
		g := gapsOf(n.Gap, len(ln.kids))
		if horiz {
			n.place(ln.kids, true, pl, pt+off, innerW, ln.cross, g)
		} else {
			n.place(ln.kids, false, pl+off, pt, ln.cross, mainAvail, g)
		}
		off += ln.cross + n.LineGap
	}
}

func (n *Node) place(kids []*Node, horiz bool, pl, pt, innerW, innerH, gaps float64) {
	used := 0.0
	for _, k := range kids {
		if horiz {
			used += k.RW
		} else {
			used += k.RH
		}
	}
	mainSize, crossSize := innerW, innerH
	if !horiz {
		mainSize, crossSize = innerH, innerW
	}
	gap, off := n.Gap, 0.0
	switch n.Justify {
	case Center:
		off = (mainSize - used - gaps) / 2
	case End:
		off = mainSize - used - gaps
	case SpaceBetween:
		if len(kids) > 1 {
			gap = (mainSize - used) / float64(len(kids)-1)
		}
	}
	for _, k := range kids {
		ks, cs := k.RH, k.RW
		if horiz {
			ks, cs = k.RW, k.RH
		}
		co := 0.0
		switch n.Align {
		case Center:
			co = (crossSize - cs) / 2
		case End:
			co = crossSize - cs
		}
		if horiz {
			k.RX, k.RY = pl+off, pt+co
		} else {
			k.RX, k.RY = pl+co, pt+off
		}
		off += ks + gap
	}
}
