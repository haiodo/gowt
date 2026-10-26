package gfx

import (
	"errors"
	"math"
	"strconv"
)

// ParseSVGPath reads the d attribute of an SVG path into a Path.
func ParseSVGPath(d string) (*Path, error) {
	sc := &scan{s: d}
	p := &Path{}
	var cur, start, lastC Point // lastC is the last control point, for the smooth commands
	var cmd, prev byte
	for {
		sc.skip()
		if sc.i >= len(sc.s) {
			break
		}
		if c := sc.s[sc.i]; isCmd(c) {
			cmd = c
			sc.i++
		} else if cmd == 0 || cmd == 'z' || cmd == 'Z' {
			return nil, errors.New("path: number without command")
		}
		rel := cmd >= 'a'
		up := cmd &^ 0x20
		pt := func(x, y float64) Point {
			if rel {
				return Point{X: cur.X + x, Y: cur.Y + y}
			}
			return Point{X: x, Y: y}
		}
		switch up {
		case 'M':
			v, ok := sc.nums(2)
			if !ok {
				return nil, errors.New("path: bad M")
			}
			cur = pt(v[0], v[1])
			start = cur
			p.MoveTo(cur.X, cur.Y)
			if rel {
				cmd = 'l'
			} else {
				cmd = 'L'
			}
		case 'L':
			v, ok := sc.nums(2)
			if !ok {
				return nil, errors.New("path: bad L")
			}
			cur = pt(v[0], v[1])
			p.LineTo(cur.X, cur.Y)
		case 'H':
			v, ok := sc.nums(1)
			if !ok {
				return nil, errors.New("path: bad H")
			}
			if rel {
				cur.X += v[0]
			} else {
				cur.X = v[0]
			}
			p.LineTo(cur.X, cur.Y)
		case 'V':
			v, ok := sc.nums(1)
			if !ok {
				return nil, errors.New("path: bad V")
			}
			if rel {
				cur.Y += v[0]
			} else {
				cur.Y = v[0]
			}
			p.LineTo(cur.X, cur.Y)
		case 'C':
			v, ok := sc.nums(6)
			if !ok {
				return nil, errors.New("path: bad C")
			}
			c1, c2, e := pt(v[0], v[1]), pt(v[2], v[3]), pt(v[4], v[5])
			p.CubicTo(c1.X, c1.Y, c2.X, c2.Y, e.X, e.Y)
			lastC, cur = c2, e
		case 'S':
			v, ok := sc.nums(4)
			if !ok {
				return nil, errors.New("path: bad S")
			}
			c1 := cur
			if prev == 'C' || prev == 'S' {
				c1 = Point{X: 2*cur.X - lastC.X, Y: 2*cur.Y - lastC.Y}
			}
			c2, e := pt(v[0], v[1]), pt(v[2], v[3])
			p.CubicTo(c1.X, c1.Y, c2.X, c2.Y, e.X, e.Y)
			lastC, cur = c2, e
		case 'Q':
			v, ok := sc.nums(4)
			if !ok {
				return nil, errors.New("path: bad Q")
			}
			c, e := pt(v[0], v[1]), pt(v[2], v[3])
			p.QuadTo(c.X, c.Y, e.X, e.Y)
			lastC, cur = c, e
		case 'T':
			v, ok := sc.nums(2)
			if !ok {
				return nil, errors.New("path: bad T")
			}
			c := cur
			if prev == 'Q' || prev == 'T' {
				c = Point{X: 2*cur.X - lastC.X, Y: 2*cur.Y - lastC.Y}
			}
			e := pt(v[0], v[1])
			p.QuadTo(c.X, c.Y, e.X, e.Y)
			lastC, cur = c, e
		case 'A':
			v, ok := sc.arc()
			if !ok {
				return nil, errors.New("path: bad A")
			}
			e := pt(v[5], v[6])
			arcTo(p, cur, v[0], v[1], v[2], v[3] != 0, v[4] != 0, e)
			cur = e
		case 'Z':
			p.Close()
			cur = start
		default:
			return nil, errors.New("path: unknown command " + string(cmd))
		}
		prev = up
	}
	return p, nil
}

func isCmd(c byte) bool {
	switch c | 0x20 {
	case 'm', 'l', 'h', 'v', 'c', 's', 'q', 't', 'a', 'z':
		return true
	}
	return false
}

type scan struct {
	s string
	i int
}

func (s *scan) skip() {
	for s.i < len(s.s) && (s.s[s.i] == ' ' || s.s[s.i] == ',' || s.s[s.i] == '\n' || s.s[s.i] == '\t' || s.s[s.i] == '\r') {
		s.i++
	}
}

func (s *scan) num() (float64, bool) {
	s.skip()
	j := s.i
	if j < len(s.s) && (s.s[j] == '-' || s.s[j] == '+') {
		j++
	}
	dot := false
	for j < len(s.s) {
		c := s.s[j]
		if c >= '0' && c <= '9' {
			j++
		} else if c == '.' && !dot {
			dot = true
			j++
		} else {
			break
		}
	}
	if j < len(s.s) && (s.s[j] == 'e' || s.s[j] == 'E') {
		k := j + 1
		if k < len(s.s) && (s.s[k] == '-' || s.s[k] == '+') {
			k++
		}
		if k < len(s.s) && s.s[k] >= '0' && s.s[k] <= '9' {
			for k < len(s.s) && s.s[k] >= '0' && s.s[k] <= '9' {
				k++
			}
			j = k
		}
	}
	v, err := strconv.ParseFloat(s.s[s.i:j], 64)
	if err != nil {
		return 0, false
	}
	s.i = j
	return v, true
}

func (s *scan) nums(n int) ([6]float64, bool) {
	var out [6]float64
	for i := 0; i < n; i++ {
		v, ok := s.num()
		if !ok {
			return out, false
		}
		out[i] = v
	}
	return out, true
}

// arc reads rx ry rotation large sweep x y; the two flags are single characters.
func (s *scan) arc() ([7]float64, bool) {
	var out [7]float64
	for i := 0; i < 7; i++ {
		if i == 3 || i == 4 {
			s.skip()
			if s.i >= len(s.s) || (s.s[s.i] != '0' && s.s[s.i] != '1') {
				return out, false
			}
			out[i] = float64(s.s[s.i] - '0')
			s.i++
			continue
		}
		v, ok := s.num()
		if !ok {
			return out, false
		}
		out[i] = v
	}
	return out, true
}

// arcTo appends the elliptical arc from p0 to p1 as cubics (SVG endpoint parameterization).
func arcTo(p *Path, p0 Point, rx, ry, rotDeg float64, large, sweep bool, p1 Point) {
	rx, ry = math.Abs(rx), math.Abs(ry)
	if rx == 0 || ry == 0 || p0 == p1 {
		p.LineTo(p1.X, p1.Y)
		return
	}
	phi := rotDeg * math.Pi / 180
	sphi, cphi := math.Sincos(phi)
	dx, dy := (p0.X-p1.X)/2, (p0.Y-p1.Y)/2
	x1, y1 := cphi*dx+sphi*dy, -sphi*dx+cphi*dy
	if l := x1*x1/(rx*rx) + y1*y1/(ry*ry); l > 1 {
		s := math.Sqrt(l)
		rx, ry = rx*s, ry*s
	}
	num := rx*rx*ry*ry - rx*rx*y1*y1 - ry*ry*x1*x1
	den := rx*rx*y1*y1 + ry*ry*x1*x1
	co := 0.0
	if den != 0 && num > 0 {
		co = math.Sqrt(num / den)
	}
	if large == sweep {
		co = -co
	}
	cxp, cyp := co*rx*y1/ry, -co*ry*x1/rx
	cx := cphi*cxp - sphi*cyp + (p0.X+p1.X)/2
	cy := sphi*cxp + cphi*cyp + (p0.Y+p1.Y)/2
	ang := func(ux, uy, vx, vy float64) float64 {
		return math.Atan2(ux*vy-uy*vx, ux*vx+uy*vy)
	}
	th1 := ang(1, 0, (x1-cxp)/rx, (y1-cyp)/ry)
	dth := ang((x1-cxp)/rx, (y1-cyp)/ry, (-x1-cxp)/rx, (-y1-cyp)/ry)
	if !sweep && dth > 0 {
		dth -= 2 * math.Pi
	} else if sweep && dth < 0 {
		dth += 2 * math.Pi
	}
	n := int(math.Ceil(math.Abs(dth) / (math.Pi / 2)))
	step := dth / float64(n)
	k := 4.0 / 3 * math.Tan(step/4)
	at := func(t float64) (Point, Point) {
		s, c := math.Sincos(t)
		pt := Point{X: cx + cphi*rx*c - sphi*ry*s, Y: cy + sphi*rx*c + cphi*ry*s}
		d := Point{X: -cphi*rx*s - sphi*ry*c, Y: -sphi*rx*s + cphi*ry*c}
		return pt, d
	}
	for i := 0; i < n; i++ {
		a0, a1 := th1+float64(i)*step, th1+float64(i+1)*step
		q0, d0 := at(a0)
		q1, d1 := at(a1)
		if i == n-1 {
			q1 = p1
		}
		p.CubicTo(q0.X+k*d0.X, q0.Y+k*d0.Y, q1.X-k*d1.X, q1.Y-k*d1.Y, q1.X, q1.Y)
	}
}
