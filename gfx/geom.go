package gfx

import "math"

type Point struct{ X, Y float64 }

func (p Point) Add(q Point) Point    { return Point{p.X + q.X, p.Y + q.Y} }
func (p Point) Sub(q Point) Point    { return Point{p.X - q.X, p.Y - q.Y} }
func (p Point) Mul(f float64) Point  { return Point{p.X * f, p.Y * f} }
func (p Point) Dot(q Point) float64  { return p.X*q.X + p.Y*q.Y }
func (p Point) Len() float64         { return math.Hypot(p.X, p.Y) }
func (p Point) Dist(q Point) float64 { return math.Hypot(p.X-q.X, p.Y-q.Y) }

func lerp(a, b Point, t float64) Point { return Point{a.X + (b.X-a.X)*t, a.Y + (b.Y-a.Y)*t} }

// Unit returns p scaled to length 1, or the zero Point for a zero vector.
func (p Point) Unit() Point {
	l := p.Len()
	if l == 0 {
		return Point{}
	}
	return Point{p.X / l, p.Y / l}
}

// Rect is an axis-aligned rectangle. Use EmptyRect as the identity of Union.
type Rect struct{ Min, Max Point }

func R(x, y, w, h float64) Rect { return Rect{Point{x, y}, Point{x + w, y + h}} }

func EmptyRect() Rect {
	return Rect{Point{math.Inf(1), math.Inf(1)}, Point{math.Inf(-1), math.Inf(-1)}}
}

func (r Rect) Empty() bool   { return r.Min.X > r.Max.X || r.Min.Y > r.Max.Y }
func (r Rect) W() float64    { return r.Max.X - r.Min.X }
func (r Rect) H() float64    { return r.Max.Y - r.Min.Y }
func (r Rect) Center() Point { return Point{(r.Min.X + r.Max.X) / 2, (r.Min.Y + r.Max.Y) / 2} }

func (r Rect) Contains(p Point) bool {
	return p.X >= r.Min.X && p.X <= r.Max.X && p.Y >= r.Min.Y && p.Y <= r.Max.Y
}

func (r Rect) Intersects(o Rect) bool {
	return r.Min.X <= o.Max.X && o.Min.X <= r.Max.X && r.Min.Y <= o.Max.Y && o.Min.Y <= r.Max.Y
}

func (r Rect) Union(o Rect) Rect {
	return Rect{Point{math.Min(r.Min.X, o.Min.X), math.Min(r.Min.Y, o.Min.Y)}, Point{math.Max(r.Max.X, o.Max.X), math.Max(r.Max.Y, o.Max.Y)}}
}

func (r Rect) AddPoint(p Point) Rect {
	return Rect{Point{math.Min(r.Min.X, p.X), math.Min(r.Min.Y, p.Y)}, Point{math.Max(r.Max.X, p.X), math.Max(r.Max.Y, p.Y)}}
}

func (r Rect) Expand(d float64) Rect {
	return Rect{Point{r.Min.X - d, r.Min.Y - d}, Point{r.Max.X + d, r.Max.Y + d}}
}

func (r Rect) Translate(d Point) Rect { return Rect{r.Min.Add(d), r.Max.Add(d)} }

// Affine is [a b c d tx ty]: x' = a*x + c*y + tx, y' = b*x + d*y + ty.
type Affine [6]float64

func Identity() Affine { return Affine{1, 0, 0, 1, 0, 0} }

func (m Affine) Apply(p Point) Point {
	return Point{m[0]*p.X + m[2]*p.Y + m[4], m[1]*p.X + m[3]*p.Y + m[5]}
}

// Mul returns the transform that applies n first, then m.
func (m Affine) Mul(n Affine) Affine {
	return Affine{
		m[0]*n[0] + m[2]*n[1], m[1]*n[0] + m[3]*n[1],
		m[0]*n[2] + m[2]*n[3], m[1]*n[2] + m[3]*n[3],
		m[0]*n[4] + m[2]*n[5] + m[4], m[1]*n[4] + m[3]*n[5] + m[5],
	}
}

func TranslateBy(x, y float64) Affine { return Affine{1, 0, 0, 1, x, y} }
func ScaleBy(x, y float64) Affine     { return Affine{x, 0, 0, y, 0, 0} }

func RotateBy(a float64) Affine {
	s, c := math.Sincos(a)
	return Affine{c, s, -s, c, 0, 0}
}

// Bounds returns the bounding box of r transformed by m.
func (m Affine) Bounds(r Rect) Rect {
	out := EmptyRect()
	for _, p := range [4]Point{r.Min, {r.Max.X, r.Min.Y}, r.Max, {r.Min.X, r.Max.Y}} {
		out = out.AddPoint(m.Apply(p))
	}
	return out
}

// Det is the determinant of the linear part.
func (m Affine) Det() float64 { return m[0]*m[3] - m[1]*m[2] }

// Scale is the factor by which m scales lengths on average: sqrt(|det|).
func (m Affine) Scale() float64 { return math.Sqrt(math.Abs(m.Det())) }

// Invert returns the inverse transform; ok is false for a singular matrix.
func (m Affine) Invert() (inv Affine, ok bool) {
	d := m.Det()
	if d == 0 || math.IsNaN(d) {
		return Identity(), false
	}
	a, b, c, dd := m[3]/d, -m[1]/d, -m[2]/d, m[0]/d
	return Affine{a, b, c, dd, -(a*m[4] + c*m[5]), -(b*m[4] + dd*m[5])}, true
}

// SameLinear reports whether m and n differ at most in translation.
func (m Affine) SameLinear(n Affine) bool {
	return m[0] == n[0] && m[1] == n[1] && m[2] == n[2] && m[3] == n[3]
}

// Color is non-premultiplied RGBA in 0..1.
type Color struct{ R, G, B, A float64 }
