// Package layout is a small flexbox-like layout for design mockups: rows and columns with gap,
// padding, alignment, justification and fixed, hug and fill sizes. It knows nothing of drawing: leaves
// that depend on content (text) report their size through two callbacks. It uses the standard library
// only, so it can be copied into another module as is.
//
// Rules:
//
//   - A node is a Row (children left to right, the default), a Column, or Abs (children at their own
//     X, Y; X and Y are ignored in the other two).
//   - A size is Fixed (a number), Hug (the content plus padding) or Fill (a share of the free main
//     axis space of the parent, the whole cross axis of the parent). Fill children of one parent share
//     the free space equally.
//   - Widths are decided top down, from the intrinsic widths of the content; heights bottom up, from
//     the widths. A Hug height, or a Fill height under a parent of unknown height, is the content's.
//   - Gap goes between children. Padding is top, right, bottom, left.
//   - Justify (main axis) is Start, Center, End or SpaceBetween; Align (cross axis) is Start, Center or End.
//   - Leaves (no children) with LeafW and LeafH set take their size from them: LeafW is the natural
//     width, LeafH(w) the height at width w (text wrapping).
//
// Not supported: wrap, min and max sizes, per-child align overrides, percentages, Fill sizes with a
// fallback value.
package layout

import "math"

type Dir uint8

const (
	Row Dir = iota
	Column
	Abs
)

type Kind uint8

const (
	Hug Kind = iota
	Fixed
	Fill
)

type Size struct {
	Kind Kind
	V    float64 // the number of a Fixed size
}

type Align uint8

const (
	Start Align = iota
	Center
	End
	SpaceBetween // Justify only
)

// Node is one box. Set the inputs, call Layout on the root, read RX, RY, RW, RH.
type Node struct {
	W, H    Size
	X, Y    float64 // position in an Abs parent
	Dir     Dir
	Gap     float64
	Pad     [4]float64 // top, right, bottom, left
	Align   Align
	Justify Align
	Kids    []*Node

	// LeafW and LeafH make a leaf depend on its content.
	LeafW func() float64
	LeafH func(w float64) float64

	// Results, relative to the parent's top left corner.
	RX, RY, RW, RH float64

	iw   float64
	iwOK bool
}

// Layout sizes n from its own W and H (Hug and Fill count as Hug at the root) and places everything below.
func (n *Node) Layout() {
	w := n.intrinsicW()
	if n.W.Kind == Fixed {
		w = n.W.V
	}
	h := -1.0
	if n.H.Kind == Fixed {
		h = n.H.V
	}
	n.layout(w, h)
	n.RX, n.RY = 0, 0
}

func (n *Node) intrinsicW() float64 {
	if n.W.Kind == Fixed {
		return n.W.V
	}
	if n.iwOK {
		return n.iw
	}
	var w float64
	switch {
	case len(n.Kids) == 0:
		if n.LeafW != nil {
			w = n.LeafW()
		}
	case n.Dir == Abs:
		for _, k := range n.Kids {
			w = math.Max(w, k.X+k.intrinsicW())
		}
	case n.Dir == Row:
		w = n.Pad[1] + n.Pad[3] + n.Gap*float64(len(n.Kids)-1)
		for _, k := range n.Kids {
			w += k.intrinsicW()
		}
	default:
		for _, k := range n.Kids {
			w = math.Max(w, k.intrinsicW())
		}
		w += n.Pad[1] + n.Pad[3]
	}
	n.iw, n.iwOK = w, true
	return w
}

func (n *Node) kidW(avail float64) float64 {
	switch n.W.Kind {
	case Fixed:
		return n.W.V
	case Fill:
		return avail
	}
	return n.intrinsicW()
}

// layout sizes n to width w and height h, h < 0 meaning as high as the content.
func (n *Node) layout(w, h float64) {
	n.RW = w
	switch {
	case len(n.Kids) == 0:
		n.RH = math.Max(h, 0)
		if h < 0 && n.LeafH != nil {
			n.RH = n.LeafH(w)
		} else if n.LeafH != nil {
			n.LeafH(w) // the leaf still wraps at w
		}
	case n.Dir == Abs:
		n.layoutAbs(w, h)
	default:
		n.layoutFlow(w, h)
	}
}

func (n *Node) layoutAbs(w, h float64) {
	maxB := 0.0
	for _, k := range n.Kids {
		kh := -1.0
		switch k.H.Kind {
		case Fixed:
			kh = k.H.V
		case Fill:
			kh = h
		}
		k.layout(k.kidW(w), kh)
		k.RX, k.RY = k.X, k.Y
		maxB = math.Max(maxB, k.RY+k.RH)
	}
	n.RH = h
	if h < 0 {
		n.RH = maxB
	}
}

func (n *Node) layoutFlow(w, h float64) {
	pt, pr, pb, pl := n.Pad[0], n.Pad[1], n.Pad[2], n.Pad[3]
	innerW := w - pl - pr
	gaps := n.Gap * float64(len(n.Kids)-1)
	innerH := h - pt - pb
	if h < 0 {
		innerH = -1
	}
	if n.Dir == Row {
		fixed, nfill := 0.0, 0
		for _, k := range n.Kids {
			if k.W.Kind == Fill {
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
		for _, k := range n.Kids {
			kh := -1.0
			switch k.H.Kind {
			case Fixed:
				kh = k.H.V
			case Fill:
				kh = innerH
			}
			k.layout(k.kidW(share), kh)
			contentH = math.Max(contentH, k.RH)
		}
		if h < 0 {
			h, innerH = contentH+pt+pb, contentH
			for _, k := range n.Kids {
				if k.H.Kind == Fill {
					k.layout(k.RW, innerH)
				}
			}
		}
		n.RH = h
		n.place(true, pl, pt, innerW, innerH, gaps)
		return
	}
	sum, nfill := 0.0, 0
	for _, k := range n.Kids {
		if k.H.Kind == Fill && innerH >= 0 {
			nfill++
			continue
		}
		kh := -1.0
		if k.H.Kind == Fixed {
			kh = k.H.V
		}
		k.layout(k.kidW(innerW), kh)
		sum += k.RH
	}
	if nfill > 0 {
		share := math.Max((innerH-sum-gaps)/float64(nfill), 0)
		for _, k := range n.Kids {
			if k.H.Kind == Fill {
				k.layout(k.kidW(innerW), share)
				sum += k.RH
			}
		}
	}
	if h < 0 {
		h = sum + gaps + pt + pb
		innerH = sum + gaps
	}
	n.RH = h
	n.place(false, pl, pt, innerW, innerH, gaps)
}

func (n *Node) place(horiz bool, pl, pt, innerW, innerH, gaps float64) {
	used := 0.0
	for _, k := range n.Kids {
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
		if len(n.Kids) > 1 {
			gap = (mainSize - used) / float64(len(n.Kids)-1)
		}
	}
	for _, k := range n.Kids {
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
