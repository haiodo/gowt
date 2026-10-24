package gfx

import (
	"math"
	"slices"
)

type HitMode uint8

const (
	HitBounds HitMode = iota // the node's bounds
	HitFill                  // inside Shape
	HitStroke                // within the tolerance of Shape
)

// Node draws Pic (local coordinates) translated by (X, Y); higher Z draws later and is hit first.
// Change a placed node only through Scene.Update so the index follows.
type Node struct {
	ID    int
	X, Y  float64
	Z     int
	Pic   *Picture
	Shape *Path // local coordinates, for HitFill and HitStroke
	Hit   HitMode

	bounds Rect
	c0, c1 [2]int32 // first and last grid cell, inclusive
	placed bool
}

// Bounds returns the node's picture bounds in scene coordinates.
func (n *Node) Bounds() Rect { return n.bounds }

// Scene is a flat set of nodes in a uniform grid; queries may run concurrently while nobody writes.
// Ceiling: one level, no parent transforms or clips; a node tree needs bounds propagation.
type Scene struct {
	world Rect
	cell  float64
	nx    int
	ny    int
	grid  [][]*Node
	nodes []*Node
}

func NewScene(world Rect, cell float64) *Scene {
	nx := max(int(math.Ceil(world.W()/cell)), 1)
	ny := max(int(math.Ceil(world.H()/cell)), 1)
	return &Scene{world: world, cell: cell, nx: nx, ny: ny, grid: make([][]*Node, nx*ny)}
}

func (s *Scene) Nodes() []*Node { return s.nodes }

func (s *Scene) cellOf(p Point) [2]int32 {
	x := int(math.Floor((p.X - s.world.Min.X) / s.cell))
	y := int(math.Floor((p.Y - s.world.Min.Y) / s.cell))
	return [2]int32{int32(min(max(x, 0), s.nx-1)), int32(min(max(y, 0), s.ny-1))}
}

func (s *Scene) insert(n *Node) {
	n.bounds = n.Pic.Bounds.Translate(Point{n.X, n.Y})
	n.c0, n.c1 = s.cellOf(n.bounds.Min), s.cellOf(n.bounds.Max)
	for y := n.c0[1]; y <= n.c1[1]; y++ {
		for x := n.c0[0]; x <= n.c1[0]; x++ {
			i := int(y)*s.nx + int(x)
			s.grid[i] = append(s.grid[i], n)
		}
	}
	n.placed = true
}

func (s *Scene) remove(n *Node) {
	for y := n.c0[1]; y <= n.c1[1]; y++ {
		for x := n.c0[0]; x <= n.c1[0]; x++ {
			i := int(y)*s.nx + int(x)
			c := s.grid[i]
			for k, m := range c {
				if m == n {
					c[k] = c[len(c)-1]
					c[len(c)-1] = nil
					s.grid[i] = c[:len(c)-1]
					break
				}
			}
		}
	}
	n.placed = false
}

// Add places n in the scene; n.Pic must be set.
func (s *Scene) Add(n *Node) {
	s.nodes = append(s.nodes, n)
	s.insert(n)
}

// Update moves n and/or swaps its picture and returns the union of its old and new bounds:
// the area whose tiles are now stale.
func (s *Scene) Update(n *Node, x, y float64, pic *Picture) Rect {
	old := n.bounds
	if n.placed {
		s.remove(n)
	}
	n.X, n.Y = x, y
	if pic != nil {
		n.Pic = pic
	}
	s.insert(n)
	return old.Union(n.bounds)
}

// Query appends the nodes whose bounds meet r to dst in draw order (Z, then ID).
func (s *Scene) Query(r Rect, dst []*Node) []*Node {
	c0, c1 := s.cellOf(r.Min), s.cellOf(r.Max)
	start := len(dst)
	for y := c0[1]; y <= c1[1]; y++ {
		for x := c0[0]; x <= c1[0]; x++ {
			for _, n := range s.grid[int(y)*s.nx+int(x)] {
				// report a node once, in the first cell it shares with the query
				if int32(max(n.c0[0], c0[0])) != x || int32(max(n.c0[1], c0[1])) != y {
					continue
				}
				if n.bounds.Intersects(r) {
					dst = append(dst, n)
				}
			}
		}
	}
	slices.SortFunc(dst[start:], func(a, b *Node) int {
		if a.Z != b.Z {
			return a.Z - b.Z
		}
		return a.ID - b.ID
	})
	return dst
}

// HitTest returns the topmost node under p. tol is the stroke tolerance and the bounds slack
// in scene units (screen pixels divided by the zoom).
func (s *Scene) HitTest(p Point, tol float64) *Node {
	r := Rect{p, p}.Expand(tol)
	var buf [32]*Node
	c := s.Query(r, buf[:0])
	for i := len(c) - 1; i >= 0; i-- {
		n := c[i]
		switch n.Hit {
		case HitBounds:
			return n
		case HitFill:
			if n.Shape.Contains(Point{p.X - n.X, p.Y - n.Y}, tol/2+0.01) {
				return n
			}
		case HitStroke:
			if h, ok := n.Shape.Nearest(Point{p.X - n.X, p.Y - n.Y}); ok && h.Dist <= tol {
				return n
			}
		}
	}
	return nil
}
