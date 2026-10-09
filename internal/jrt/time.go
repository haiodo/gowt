package jrt

import "time"

// Instant and Duration are java.time's, over time.Time and time.Duration.
type Instant struct{ t time.Time }

func InstantNow() *Instant { return &Instant{time.Now()} }

func (i *Instant) PlusMillis(n int64) *Instant {
	return &Instant{i.t.Add(time.Duration(n) * time.Millisecond)}
}
func (i *Instant) PlusSeconds(n int64) *Instant {
	return &Instant{i.t.Add(time.Duration(n) * time.Second)}
}
func (i *Instant) IsBefore(o *Instant) bool { return i.t.Before(o.t) }
func (i *Instant) IsAfter(o *Instant) bool  { return i.t.After(o.t) }

type Duration struct{ d time.Duration }

func DurationOfSeconds(n int64) *Duration       { return &Duration{time.Duration(n) * time.Second} }
func DurationBetween(a, b *Instant) *Duration   { return &Duration{b.t.Sub(a.t)} }
func (d *Duration) Minus(o *Duration) *Duration { return &Duration{d.d - o.d} }
func (d *Duration) IsNegative() bool            { return d.d < 0 }
func (d *Duration) ToMillis() int64             { return d.d.Milliseconds() }
func (d *Duration) ToNanos() int64              { return d.d.Nanoseconds() }
