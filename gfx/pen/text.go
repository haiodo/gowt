package pen

import (
	"math"
	"strings"

	"github.com/haiodo/gowt/gfx"
)

type tline struct {
	s string
	w float64
}

// Counters count the work of the layout, for the benchmarks.
type Counters struct {
	Measures int // MeasureText calls
}

type layouter struct {
	d   *Doc
	cnt *Counters
}

func (l *layouter) font(n *rn) gfx.Font {
	return gfx.Font{Name: l.d.o.FontName(n.family, n.weight), Size: n.fsize}
}

func (l *layouter) measure(s string, f gfx.Font) gfx.TextMetrics {
	l.cnt.Measures++
	return l.d.o.Measurer.MeasureText(s, f)
}

func (l *layouter) lineH(n *rn) float64 {
	// pen.dev rounds the line box to whole units: 14 * 1.25 is 18, 13 * 1.25 is 16. Without a
	// lineHeight it is the font's own: rounded ascent plus rounded descent.
	if n.lh > 0 {
		return math.Round(n.lh * n.fsize)
	}
	m := l.measure("Hg", l.font(n))
	return math.Round(m.Ascent) + math.Round(m.Descent)
}

func (l *layouter) natural(n *rn) []tline {
	f := l.font(n)
	parts := strings.Split(n.content, "\n")
	out := make([]tline, len(parts))
	for i, p := range parts {
		out[i] = tline{p, l.measure(p, f).Width}
	}
	return out
}

// wrap breaks the text greedily at spaces to fit w.
func (l *layouter) wrap(n *rn, w float64) []tline {
	f := l.font(n)
	sp := l.measure(" ", f).Width
	var out []tline
	for _, para := range strings.Split(n.content, "\n") {
		line, lw := "", 0.0
		for _, word := range strings.Split(para, " ") {
			ww := l.measure(word, f).Width
			switch {
			case line == "":
				line, lw = word, ww
			case lw+sp+ww <= w:
				line, lw = line+" "+word, lw+sp+ww
			default:
				out = append(out, tline{line, lw})
				line, lw = word, ww
			}
		}
		out = append(out, tline{line, lw})
	}
	return out
}

// wire gives the text nodes of the tree their size callbacks.
func (l *layouter) wire(n *rn) {
	if n.typ == "text" {
		n.LeafW = func() float64 {
			w := 0.0
			for _, ln := range l.natural(n) {
				w = max(w, ln.w)
			}
			return math.Ceil(w)
		}
		n.LeafH = func(w float64) float64 {
			if n.growW && w > 0 {
				n.lines = l.wrap(n, w)
			} else {
				n.lines = l.natural(n)
			}
			return float64(len(n.lines)) * l.lineH(n)
		}
	}
	for _, k := range n.kids {
		l.wire(k)
	}
}
