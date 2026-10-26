package gfx

import (
	"math"
	"slices"
	"sort"
	"sync"
)

// HitMode says what part of a node a hit test accepts.
type HitMode uint8

const (
	HitBounds HitMode = iota // the node's bounds
	HitFill                  // inside the shape
	HitStroke                // within the tolerance of the shape
	HitNone                  // never a hit
)

// ClipKind is the shape of a node clip.
type ClipKind uint8

// The kinds of node clip.
const (
	ClipNone ClipKind = iota
	ClipRect
	ClipRRect
	ClipPath
)

// Node is one element of the scene tree. Its picture is drawn in the node's own coordinates:
// the transform maps them into the parent's, and the parent's into the scene's. A clip applies to
// the node's picture and to the whole subtree. Children draw after their parent, ordered by Z and
// then by the order they were added; a higher Z draws later and is hit first.
//
// A node changes only through Scene.Update; the getters read the state as of the last Commit and are
// safe on the goroutine that commits, or inside Scene.Read.
type Node struct {
	id      int
	parent  *Node
	kids    []*Node
	t       Affine
	opacity float64
	z       int
	pic     *Picture
	shape   *Path // local coordinates, for HitFill and HitStroke
	clip    *nodeClip
	hit     HitMode
	hasClip bool // a clip of an ancestor or of this node applies
	indexed bool

	// derived by Commit
	world    Affine
	alpha    float64 // opacity times that of every ancestor
	bounds   Rect    // visible bounds of the picture in scene coordinates
	cbox     *Rect   // scene area left by the clips of the ancestors and this node; nil when unclipped
	order    int32
	depth    int32
	lifted   int32 // lifted ancestors plus this node's own lift count
	liftSelf int32
	rev      uint32 // bumps when the node's own look changes other than by a plain move
	deep     uint32 // bumps when something below the node changes
	stamp    uint32
	c0, c1   [2]int32
	scene    *Scene
}

type nodeClip struct {
	kind ClipKind
	rect Rect // local
	rad  float64
	path *Path // local
}

func (n *Node) ck() ClipKind {
	if n.clip == nil {
		return ClipNone
	}
	return n.clip.kind
}

// NewNode returns a detached node. id is the key for Scene.Node; 0 leaves the node unnamed, and
// with a repeated id Scene.Node returns the last one added.
func NewNode(id int) *Node {
	return &Node{id: id, t: Identity(), opacity: 1, world: Identity(), alpha: 1, bounds: EmptyRect()}
}

// ID returns the id given to NewNode.
func (n *Node) ID() int { return n.id }

// Parent returns the parent node, the scene root for a top level node.
func (n *Node) Parent() *Node { return n.parent }

// Children returns the child nodes in paint order.
func (n *Node) Children() []*Node { return n.kids }

// Transform returns the transform to the parent as of the last Commit.
func (n *Node) Transform() Affine { return n.t }

// World returns the transform to scene coordinates as of the last Commit.
func (n *Node) World() Affine { return n.world }

// Opacity returns the node's own opacity.
func (n *Node) Opacity() float64 { return n.opacity }

// Z returns the order among siblings.
func (n *Node) Z() int { return n.z }

// Picture returns the picture of the node, or nil.
func (n *Node) Picture() *Picture { return n.pic }

// Shape returns the hit shape, or nil.
func (n *Node) Shape() *Path { return n.shape }

// Hit returns the hit mode.
func (n *Node) Hit() HitMode { return n.hit }

// Lifted reports whether the node, or an ancestor, is on the active layer.
func (n *Node) Lifted() bool { return n.lifted > 0 }

// Version returns counters that change when the node, or something below it, changes; for caches.
func (n *Node) Version() (self, deep uint32) { return n.rev, n.deep }

// Bounds is the visible area of the node's own picture in scene coordinates, after the clips of
// the ancestors; empty when the node has no picture or is clipped away.
func (n *Node) Bounds() Rect { return n.bounds }

// SubtreeBounds is the union of Bounds over the node and everything below it.
func (n *Node) SubtreeBounds() Rect {
	r := n.bounds
	for _, c := range n.kids {
		r = r.Union(c.SubtreeBounds())
	}
	return r
}

// Scene is a tree of nodes with a uniform grid over the visible bounds of their pictures.
// Changes are made in transactions with Update and applied by Commit, which the frame loop
// calls; queries may run concurrently with each other and with Update.
type Scene struct {
	mu    sync.RWMutex
	root  *Node
	grid  map[[2]int32][]*Node
	big   []*Node // nodes that span more than bigSpan cells on a side
	ids   map[int]*Node
	pass  uint32
	count int

	structDirty bool

	pmu     sync.Mutex
	pending []*Tx
}

// NewScene returns an empty scene. It has no bounds: nodes may lie anywhere.
func NewScene() *Scene {
	s := &Scene{grid: map[[2]int32][]*Node{}, ids: map[int]*Node{}}
	s.root = NewNode(0)
	s.root.scene = s
	return s
}

// Root is the node every top level node hangs under; it has no picture.
func (s *Scene) Root() *Node { return s.root }

// Len is the number of nodes in the scene, the root excluded.
func (s *Scene) Len() int { s.mu.RLock(); defer s.mu.RUnlock(); return s.count }

// Node returns the node with the given ID, or nil.
func (s *Scene) Node(id int) *Node { s.mu.RLock(); defer s.mu.RUnlock(); return s.ids[id] }

// Read runs f holding the read lock, so that the getters of nodes are safe on any goroutine.
func (s *Scene) Read(f func()) { s.mu.RLock(); defer s.mu.RUnlock(); f() }

type txKind uint8

const (
	txAdd txKind = iota
	txRemove
	txTransform
	txOpacity
	txZ
	txPicture
	txShape
	txClip
	txReparent
	txSuperseded // a txPicture followed by another one for the same node in this Commit
)

type txOp struct {
	kind txKind
	n, p *Node
	m    Affine
	f    float64
	i    int
	pic  *Picture
	path *Path
	r    Rect
	ck   ClipKind
}

// Tx records changes to a scene. It is valid only inside the Update call that made it.
type Tx struct{ ops []txOp }

// Update runs f to record a transaction and queues it; Commit applies it whole, so a frame
// shows either all of it or none. It is safe from any goroutine, and f runs before Update returns.
func (s *Scene) Update(f func(tx *Tx)) {
	tx := &Tx{}
	f(tx)
	if len(tx.ops) == 0 {
		return
	}
	s.pmu.Lock()
	s.pending = append(s.pending, tx)
	s.pmu.Unlock()
}

// Pending reports whether a transaction waits for Commit.
func (s *Scene) Pending() bool {
	s.pmu.Lock()
	defer s.pmu.Unlock()
	return len(s.pending) > 0
}

// Add attaches n, with its subtree, under parent; a nil parent means the root.
func (tx *Tx) Add(parent, n *Node) { tx.ops = append(tx.ops, txOp{kind: txAdd, n: n, p: parent}) }

// Remove detaches n with its subtree and forgets the IDs.
func (tx *Tx) Remove(n *Node) { tx.ops = append(tx.ops, txOp{kind: txRemove, n: n}) }

// Reparent moves n under another parent, keeping its own transform.
func (tx *Tx) Reparent(n, parent *Node) {
	tx.ops = append(tx.ops, txOp{kind: txReparent, n: n, p: parent})
}

// SetTransform sets the transform of n relative to its parent.
func (tx *Tx) SetTransform(n *Node, m Affine) {
	tx.ops = append(tx.ops, txOp{kind: txTransform, n: n, m: m})
}

// Move sets the translation of n, keeping the rest of its transform.
func (tx *Tx) Move(n *Node, x, y float64) {
	tx.ops = append(tx.ops, txOp{kind: txTransform, n: n, i: 1, m: Affine{0, 0, 0, 0, x, y}})
}

// SetOpacity sets the opacity of n, 0 to 1; it multiplies down the subtree.
func (tx *Tx) SetOpacity(n *Node, v float64) {
	tx.ops = append(tx.ops, txOp{kind: txOpacity, n: n, f: v})
}

// SetZ sets the order of n among its siblings; a higher Z draws later and is hit first.
func (tx *Tx) SetZ(n *Node, z int) { tx.ops = append(tx.ops, txOp{kind: txZ, n: n, i: z}) }

// SetPicture sets what n draws, in its own coordinates. A picture equal to the current one changes
// nothing.
func (tx *Tx) SetPicture(n *Node, pic *Picture) {
	tx.ops = append(tx.ops, txOp{kind: txPicture, n: n, pic: pic})
}

// SetShape sets the geometry hit-testing uses (local coordinates, kept by reference) and how.
func (tx *Tx) SetShape(n *Node, shape *Path, hit HitMode) {
	tx.ops = append(tx.ops, txOp{kind: txShape, n: n, path: shape, i: int(hit)})
}

// ClipRect clips n and its subtree to r (in n's coordinates).
func (tx *Tx) ClipRect(n *Node, r Rect) {
	tx.ops = append(tx.ops, txOp{kind: txClip, n: n, ck: ClipRect, r: r})
}

// ClipRRect clips n and its subtree to a rounded rectangle in the coordinates of n.
func (tx *Tx) ClipRRect(n *Node, r Rect, radius float64) {
	tx.ops = append(tx.ops, txOp{kind: txClip, n: n, ck: ClipRRect, r: r, f: radius})
}

// ClipRRect4 is ClipRRect with a radius per corner (top left, top right, bottom right, bottom left).
func (tx *Tx) ClipRRect4(n *Node, r Rect, radii [4]float64) { tx.ClipPath(n, RoundRectPath4(r, radii)) }

// ClipPath keeps p by reference.
func (tx *Tx) ClipPath(n *Node, p *Path) {
	tx.ops = append(tx.ops, txOp{kind: txClip, n: n, ck: ClipPath, path: p})
}

// NoClip removes the clip of n.
func (tx *Tx) NoClip(n *Node) { tx.ops = append(tx.ops, txOp{kind: txClip, n: n, ck: ClipNone}) }

// Commit applies the queued transactions in order and returns the scene areas whose pixels
// changed: the old and the new bounds of every node that moved or changed, lifted nodes aside.
// A window's frame loop calls it. Without one (a test, a program that renders to PNG) call it after
// Update before reading nodes or hit-testing: until then HitTest, Query and the getters of Node
// still show the scene as it was. RenderSubtree and offscreen.PNG commit by themselves.
func (s *Scene) Commit() []Rect {
	s.pmu.Lock()
	txs := s.pending
	s.pending = nil
	s.pmu.Unlock()
	if len(txs) == 0 {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var dirty []*Node
	var damage []Rect
	// only the last picture of a node counts, so that a node set to an intermediate picture and back
	// within one Commit (a connector between two boxes moved one after the other) is not changed
	last := map[*Node]struct{}{}
	for i := len(txs) - 1; i >= 0; i-- {
		ops := txs[i].ops
		for k := len(ops) - 1; k >= 0; k-- {
			if ops[k].kind != txPicture {
				continue
			}
			if _, dup := last[ops[k].n]; dup {
				ops[k].kind = txSuperseded
			}
			last[ops[k].n] = struct{}{}
		}
	}
	for _, tx := range txs {
		for i := range tx.ops {
			s.apply(&tx.ops[i], &dirty, &damage)
		}
	}
	// parents first, so that a node refreshed under a refreshed ancestor is not done twice
	slices.SortStableFunc(dirty, func(a, b *Node) int { return int(a.depth - b.depth) })
	s.pass++
	for _, n := range dirty {
		if n.stamp != s.pass && n.scene == s {
			s.refresh(n, &damage)
		}
	}
	if s.structDirty {
		s.structDirty = false
		s.number()
	}
	return damage
}

func (s *Scene) attach(parent, n *Node, dirty *[]*Node) {
	if parent == nil {
		parent = s.root
	}
	n.parent = parent
	i := sort.Search(len(parent.kids), func(i int) bool { return parent.kids[i].z > n.z })
	parent.kids = slices.Insert(parent.kids, i, n)
	s.bump(parent)
	s.depthOf(n)
	s.adopt(n)
	s.structDirty = true
	*dirty = append(*dirty, n)
}

func (s *Scene) adopt(n *Node) {
	n.scene = s
	s.count++
	if n.id != 0 {
		s.ids[n.id] = n
	}
	for _, c := range n.kids {
		s.adopt(c)
	}
}

func (s *Scene) depthOf(n *Node) {
	n.depth = n.parent.depth + 1
	for _, c := range n.kids {
		s.depthOf(c)
	}
}

// bump marks every ancestor of n as changed below.
func (s *Scene) bump(n *Node) {
	for ; n != nil; n = n.parent {
		n.deep++
	}
}

func (s *Scene) detach(n *Node, damage *[]Rect) {
	if n.parent != nil {
		p := n.parent
		if i := slices.Index(p.kids, n); i >= 0 {
			p.kids = slices.Delete(p.kids, i, i+1)
		}
		s.bump(p)
		n.parent = nil
	}
	s.forget(n, damage)
	s.structDirty = true
}

func (s *Scene) forget(n *Node, damage *[]Rect) {
	if n.indexed {
		if n.lifted == 0 {
			*damage = append(*damage, n.bounds)
		}
		s.remove(n)
	}
	if n.scene == s {
		s.count--
	}
	if n.id != 0 && s.ids[n.id] == n {
		delete(s.ids, n.id)
	}
	n.scene = nil
	for _, c := range n.kids {
		s.forget(c, damage)
	}
}

func (s *Scene) apply(op *txOp, dirty *[]*Node, damage *[]Rect) {
	n := op.n
	if n == nil {
		return
	}
	if (op.kind == txRemove || op.kind == txReparent) && n.scene != s {
		return // removed, or never added
	}
	switch op.kind {
	case txAdd:
		if n.scene == s || n == s.root {
			return
		}
		s.attach(op.p, n, dirty)
	case txRemove:
		if n != s.root {
			s.detach(n, damage)
		}
	case txReparent:
		if n != s.root {
			p := op.p
			if p == nil {
				p = s.root
			}
			for q := p; q != nil; q = q.parent {
				if q == n {
					return // into its own subtree
				}
			}
			old := n.parent
			if i := slices.Index(old.kids, n); i >= 0 {
				old.kids = slices.Delete(old.kids, i, i+1)
			}
			s.bump(old)
			n.parent = p
			i := sort.Search(len(p.kids), func(i int) bool { return p.kids[i].z > n.z })
			p.kids = slices.Insert(p.kids, i, n)
			s.bump(p)
			s.depthOf(n)
			n.rev++
			s.structDirty = true
			*dirty = append(*dirty, n)
		}
	case txTransform:
		m := op.m
		if op.i == 1 {
			m = n.t
			m[4], m[5] = op.m[4], op.m[5]
		}
		if m == n.t {
			return
		}
		if !m.SameLinear(n.t) {
			n.rev++
		}
		n.t = m
		s.bump(n.parent)
		*dirty = append(*dirty, n)
	case txOpacity:
		if op.f != n.opacity {
			n.opacity = op.f
			n.rev++
			s.bump(n.parent)
			*dirty = append(*dirty, n)
		}
	case txZ:
		if op.i != n.z {
			n.z = op.i
			if p := n.parent; p != nil {
				i := slices.Index(p.kids, n)
				p.kids = slices.Delete(p.kids, i, i+1)
				j := sort.Search(len(p.kids), func(k int) bool { return p.kids[k].z > n.z })
				p.kids = slices.Insert(p.kids, j, n)
				s.bump(p)
			}
			s.structDirty = true
		}
	case txPicture:
		if n.pic.Equal(op.pic) {
			return
		}
		n.pic = op.pic
		n.rev++
		s.bump(n.parent)
		*dirty = append(*dirty, n)
	case txShape:
		n.shape, n.hit = op.path, HitMode(op.i)
	case txClip:
		n.clip = nil
		if op.ck != ClipNone {
			n.clip = &nodeClip{op.ck, op.r, op.f, op.path}
		}
		n.rev++
		s.bump(n.parent)
		*dirty = append(*dirty, n)
	}
}

// refresh recomputes the derived state of n and its subtree and reindexes what moved.
func (s *Scene) refresh(n *Node, damage *[]Rect) {
	n.stamp = s.pass
	p := n.parent
	n.world = p.world.Mul(n.t)
	n.alpha = p.alpha * n.opacity
	n.lifted = p.lifted + n.liftSelf
	n.cbox = p.cbox
	n.hasClip = p.hasClip
	switch n.ck() {
	case ClipRect, ClipRRect, ClipPath:
		local := n.clip.rect
		if n.clip.kind == ClipPath {
			local = n.clip.path.Bounds()
		}
		box := n.world.Bounds(local)
		if p.cbox != nil {
			box = box.Intersect(*p.cbox)
		}
		n.cbox = &box
		n.hasClip = true
	}
	old, wasIndexed := n.bounds, n.indexed
	nb := EmptyRect()
	if n.pic != nil && !n.pic.Bounds.Empty() { // a picture that draws nothing has no bounds to map
		nb = n.world.Bounds(n.pic.Bounds)
		if n.cbox != nil {
			nb = nb.Intersect(*n.cbox)
		}
	}
	if wasIndexed {
		s.remove(n)
	}
	n.bounds = nb
	if !nb.Empty() {
		s.insert(n)
	}
	if n.lifted == 0 {
		if wasIndexed {
			*damage = append(*damage, old)
		}
		if n.indexed {
			*damage = append(*damage, nb)
		}
	}
	for _, c := range n.kids {
		s.refresh(c, damage)
	}
}

// number assigns the paint order: depth first, a node before its children.
func (s *Scene) number() {
	i := 0
	var walk func(n *Node)
	walk = func(n *Node) {
		n.order = int32(i)
		i++
		for _, c := range n.kids {
			walk(c)
		}
	}
	walk(s.root)
}

const (
	cellSize = 256 // scene units per grid cell
	cellMax  = 1 << 30
	// bigSpan is the side, in cells, above which a node is kept in one list instead of in every cell
	// it covers, so that a backdrop the size of the world does not fill the grid.
	bigSpan = 32
)

func cellOf(p Point) [2]int32 {
	f := func(v float64) int32 { return int32(min(max(math.Floor(v/cellSize), -cellMax), cellMax)) }
	return [2]int32{f(p.X), f(p.Y)}
}

func (s *Scene) insert(n *Node) {
	n.c0, n.c1 = cellOf(n.bounds.Min), cellOf(n.bounds.Max)
	n.indexed = true
	if n.c1[0]-n.c0[0] >= bigSpan || n.c1[1]-n.c0[1] >= bigSpan {
		s.big = append(s.big, n)
		return
	}
	for y := n.c0[1]; y <= n.c1[1]; y++ {
		for x := n.c0[0]; x <= n.c1[0]; x++ {
			k := [2]int32{x, y}
			s.grid[k] = append(s.grid[k], n)
		}
	}
}

func (s *Scene) remove(n *Node) {
	n.indexed = false
	if n.c1[0]-n.c0[0] >= bigSpan || n.c1[1]-n.c0[1] >= bigSpan {
		s.big = slices.DeleteFunc(s.big, func(m *Node) bool { return m == n })
		return
	}
	for y := n.c0[1]; y <= n.c1[1]; y++ {
		for x := n.c0[0]; x <= n.c1[0]; x++ {
			k := [2]int32{x, y}
			c := s.grid[k]
			for i, m := range c {
				if m == n {
					c[i] = c[len(c)-1]
					c[len(c)-1] = nil
					if c = c[:len(c)-1]; len(c) == 0 {
						delete(s.grid, k)
					} else {
						s.grid[k] = c
					}
					break
				}
			}
		}
	}
}

// liftNode marks n and its subtree as drawn by the compositor's active layer: Collect leaves them
// out and their changes make no damage. It returns the area the static tiles must redraw.
func (s *Scene) liftNode(n *Node) Rect { return s.liftBy(n, 1) }

// landNode undoes liftNode and returns the area the static tiles must redraw now.
func (s *Scene) landNode(n *Node) Rect { return s.liftBy(n, -1) }

func (s *Scene) liftBy(n *Node, d int32) Rect {
	s.mu.Lock()
	defer s.mu.Unlock()
	if d < 0 && n.liftSelf == 0 {
		return EmptyRect()
	}
	n.liftSelf += d
	var walk func(m *Node)
	walk = func(m *Node) {
		m.lifted += d
		for _, c := range m.kids {
			walk(c)
		}
	}
	walk(n)
	return n.SubtreeBounds()
}

// Query appends the nodes whose visible bounds meet r to dst in paint order.
func (s *Scene) Query(r Rect, dst []*Node) []*Node {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.query(r, dst)
}

func (s *Scene) query(r Rect, dst []*Node) []*Node {
	c0, c1 := cellOf(r.Min), cellOf(r.Max)
	start := len(dst)
	// report a node once, in the first cell it shares with the query
	visit := func(x, y int32, cell []*Node) {
		for _, n := range cell {
			if max(n.c0[0], c0[0]) == x && max(n.c0[1], c0[1]) == y && n.bounds.Intersects(r) {
				dst = append(dst, n)
			}
		}
	}
	if span := int64(c1[0]-c0[0]+1) * int64(c1[1]-c0[1]+1); span > int64(len(s.grid)) {
		for k, cell := range s.grid { // fewer occupied cells than cells in the query
			if k[0] >= c0[0] && k[0] <= c1[0] && k[1] >= c0[1] && k[1] <= c1[1] {
				visit(k[0], k[1], cell)
			}
		}
	} else {
		for y := c0[1]; y <= c1[1]; y++ {
			for x := c0[0]; x <= c1[0]; x++ {
				visit(x, y, s.grid[[2]int32{x, y}])
			}
		}
	}
	for _, n := range s.big {
		if n.bounds.Intersects(r) {
			dst = append(dst, n)
		}
	}
	slices.SortFunc(dst[start:], func(a, b *Node) int { return int(a.order - b.order) })
	return dst
}

// clipChain appends the clips of n and its ancestors, outermost first, in scene coordinates.
func clipChain(n *Node, dst []Clip) []Clip {
	start := len(dst)
	for ; n != nil; n = n.parent {
		if n.ck() != ClipNone {
			dst = append(dst, Clip{M: n.world, Kind: n.ck(), Rect: n.clip.rect, Radius: n.clip.rad, Path: n.clip.path})
		}
	}
	slices.Reverse(dst[start:])
	return dst
}

// Items is a reusable buffer for Collect.
type Items struct {
	List  []Item
	clips []Clip
	nodes []*Node
}

// Reset empties b, keeping its memory.
func (b *Items) Reset() { b.List, b.clips, b.nodes = b.List[:0], b.clips[:0], b.nodes[:0] }

func (b *Items) add(n *Node) {
	it := Item{Pic: n.pic, M: n.world, Alpha: n.alpha}
	if n.hasClip {
		i := len(b.clips)
		b.clips = clipChain(n, b.clips)
		it.Clips = b.clips[i:len(b.clips):len(b.clips)]
	}
	b.List = append(b.List, it)
}

// Collect fills b with what is drawn inside r, in paint order, leaving out lifted nodes.
// The items are valid until the next Collect into b, and need no lock to draw.
func (s *Scene) Collect(r Rect, b *Items) {
	b.Reset()
	s.mu.RLock()
	defer s.mu.RUnlock()
	b.nodes = s.query(r, b.nodes)
	for _, n := range b.nodes {
		if n.lifted == 0 {
			b.add(n)
		}
	}
}

// CollectSubtree is Collect for root and its descendants only, lifted or not.
func (s *Scene) CollectSubtree(root *Node, r Rect, b *Items) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	collectSubtree(root, r, b)
}

// collectSubtree is CollectSubtree for a caller that holds the scene lock.
func collectSubtree(root *Node, r Rect, b *Items) {
	b.Reset()
	var walk func(n *Node)
	walk = func(n *Node) {
		if n.indexed && n.bounds.Intersects(r) {
			b.add(n)
		}
		for _, c := range n.kids {
			walk(c)
		}
	}
	walk(root)
}

// inClip reports whether the scene point p is inside every clip of n and its ancestors.
func inClip(n *Node, p Point) bool {
	for ; n != nil; n = n.parent {
		if n.ck() == ClipNone {
			continue
		}
		inv, ok := n.world.Invert()
		if !ok {
			return false
		}
		q := inv.Apply(p)
		switch n.ck() {
		case ClipRect:
			if !n.clip.rect.Contains(q) {
				return false
			}
		case ClipRRect:
			var rr Path
			rr.RoundRect(n.clip.rect.Min.X, n.clip.rect.Min.Y, n.clip.rect.W(), n.clip.rect.H(), n.clip.rad)
			if !rr.Contains(q, 0) {
				return false
			}
		case ClipPath:
			if !n.clip.path.Contains(q, 0) {
				return false
			}
		}
	}
	return true
}

// HitTest returns the topmost node under the scene point p, or nil. tol is the stroke tolerance and
// the bounds slack in scene units (screen pixels divided by the zoom). Clips of the ancestors count.
// It sees the scene as of the last Commit.
func (s *Scene) HitTest(p Point, tol float64) *Node {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var buf [32]*Node
	c := s.query(Rect{p, p}.Expand(tol), buf[:0])
	for i := len(c) - 1; i >= 0; i-- {
		n := c[i]
		if n.hit == HitNone {
			continue
		}
		if n.hasClip && !inClip(n, p) {
			continue
		}
		inv, ok := n.world.Invert()
		if !ok {
			continue
		}
		q := inv.Apply(p)
		lt := tol * inv.Scale()
		switch {
		case n.hit == HitBounds || n.shape == nil:
			return n
		case n.hit == HitFill:
			if n.shape.Contains(q, lt/2+0.01) {
				return n
			}
		case n.hit == HitStroke:
			if h, ok := n.shape.Nearest(q); ok && h.Dist <= lt {
				return n
			}
		}
	}
	return nil
}
