package rx

// Reader is what bindings take: a State or a Derived.
type Reader[T any] interface {
	Get() T
	Peek() T
}

// State is a writable signal. Its zero value is not usable; make one with New or NewFunc.
type State[T any] struct {
	node
	v  T
	eq func(a, b T) bool

	// Post's mailbox; guarded by postMu.
	posted  bool
	pending T
}

func eqComparable[T comparable](a, b T) bool { return a == b }

// New makes a State compared with ==: a Set of an equal value does nothing.
func New[T comparable](v T) *State[T] { return NewFunc(v, eqComparable[T]) }

// NewFunc makes a State with a custom equality; eq nil means every Set notifies, which suits
// slices and other values that are replaced, not mutated.
func NewFunc[T any](v T, eq func(a, b T) bool) *State[T] {
	return &State[T]{node: node{kind: kindState}, v: v, eq: eq}
}

// Get returns the value and subscribes the Derived or Effect that is running.
func (s *State[T]) Get() T {
	if checkReads {
		mustUI("State.Get")
	}
	track(&s.node)
	return s.v
}

// Peek returns the value without subscribing.
func (s *State[T]) Peek() T { return s.v }

// Set stores v. Observers are marked now, so a Derived read right after sees v; effects run at
// the next Flush. Set panics off the UI thread (use Post) and inside a Derived.
func (s *State[T]) Set(v T) {
	mustUI("State.Set")
	// owner, not observer: an unobserved Derived computes untracked, and Untrack must not hide the write.
	if o := owner; o != nil && o.kind == kindDerived {
		panic("rx: State.Set inside a Derived; a Derived computes a value, an Effect writes")
	}
	if s.eq != nil && s.eq(s.v, v) {
		return
	}
	s.v = v
	epoch++
	s.changed = epoch
	for _, ob := range s.observers {
		ob.mark(dirty)
	}
}

// Update sets fn(current value).
func (s *State[T]) Update(fn func(T) T) { s.Set(fn(s.v)) }

// Derived is a lazy memoized computation over other signals. It recomputes when read after a
// source changed; an equal result stops the change from reaching its observers.
type Derived[T any] struct {
	node
	v    T
	fn   func() T
	eq   func(a, b T) bool
	done bool // computed at least once; the zero v is not a previous value
}

// Compute makes a Derived compared with ==.
func Compute[T comparable](fn func() T) *Derived[T] { return ComputeFunc(fn, eqComparable[T]) }

// ComputeFunc makes a Derived with a custom equality; eq nil means every recompute propagates.
// Inside an Effect or a Scope the Derived is owned by it and stops updating when disposed.
func ComputeFunc[T any](fn func() T, eq func(a, b T) bool) *Derived[T] {
	mustUI("Compute")
	d := &Derived[T]{node: node{kind: kindDerived, state: dirty}, fn: fn, eq: eq}
	d.exec = d.recompute
	d.adopt()
	return d
}

func (d *Derived[T]) recompute() {
	d.stale = true // a panicking cleanup must not leave an old value marked clean; execute clears it
	d.release()
	var v T
	d.execute(func() { v = d.fn() })
	d.verified = epoch
	if d.done && d.eq != nil && d.eq(d.v, v) {
		return
	}
	d.v, d.done, d.changed = v, true, epoch
	for _, o := range d.observers {
		if !o.running { // a running observer is reading the new value right now
			o.mark(dirty)
		}
	}
}

// Get returns the value, recomputing it if a source changed, and subscribes the running
// Derived or Effect.
func (d *Derived[T]) Get() T {
	if checkReads {
		mustUI("Derived.Get")
	}
	if d.running {
		panic("rx: Derived reads itself (dependency cycle)")
	}
	if d.disposed {
		track(&d.node)
		return d.v
	}
	if observer == nil && len(d.observers) == 0 {
		return d.getUnobserved()
	}
	track(&d.node)
	d.update()
	return d.v
}

// getUnobserved serves a read nobody tracks: the value is computed without subscribing, so a
// throwaway Derived does not live as long as its sources, and reused until the next Set.
func (d *Derived[T]) getUnobserved() T {
	if d.done && !d.stale && d.verified == epoch {
		return d.v
	}
	if len(d.sources) > 0 {
		d.unlink()
	}
	d.state, d.untracked = dirty, true
	defer func() { d.untracked = false }() // release may panic before execute consumes it
	d.update()
	d.state = dirty // nothing marks it now: the next read checks the epoch, a tracked one subscribes
	return d.v
}

// Peek is Get without subscribing.
func (d *Derived[T]) Peek() T {
	prev := observer
	observer = nil
	defer func() { observer = prev }()
	return d.Get()
}
