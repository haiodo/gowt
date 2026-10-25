package gfx

import "testing"

func benchCurve() *Path {
	var p Path
	p.MoveTo(0, 0)
	p.CubicTo(120, 0, 80, 100, 200, 100)
	return &p
}

func BenchmarkBounds(b *testing.B) {
	p := benchCurve()
	for i := 0; i < b.N; i++ {
		p.Bounds()
	}
}

func BenchmarkFlatten(b *testing.B) {
	p := benchCurve()
	for i := 0; i < b.N; i++ {
		p.Flatten(0.1)
	}
}

func BenchmarkMeasureAt(b *testing.B) {
	p := benchCurve()
	b.Run("measure", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			p.Measure(0.05)
		}
	})
	b.Run("at", func(b *testing.B) {
		m := p.Measure(0.05)
		l := m.Length()
		for i := 0; i < b.N; i++ {
			m.At(l * float64(i%100) / 100)
		}
	})
}

func BenchmarkNearest(b *testing.B) {
	p := benchCurve()
	for i := 0; i < b.N; i++ {
		p.Nearest(Point{float64(i % 200), 40})
	}
}

func BenchmarkSplit(b *testing.B) {
	p := benchCurve()
	for i := 0; i < b.N; i++ {
		p.Split(0.37)
	}
}
