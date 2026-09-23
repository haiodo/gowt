// Hand-written: assertions ported from the SWT JUnit tests for
// org.eclipse.swt.graphics.{Point,Rectangle,RGB,RGBA}.
package swt

import "testing"

func assertPanics(t *testing.T, f func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatal("expected a panic, got none")
		}
	}()
	f()
}

// --- Point (Test_org_eclipse_swt_graphics_Point) ---

func TestPointConstructor(t *testing.T) {
	p := NewPoint(3, 4)
	if p.X != 3 || p.Y != 4 {
		t.Fatalf("got {%d, %d}", p.X, p.Y)
	}
	p = NewPoint(-4, -3)
	if p.X != -4 || p.Y != -3 {
		t.Fatalf("got {%d, %d}", p.X, p.Y)
	}
}

func TestPointEquals(t *testing.T) {
	p1 := NewPoint(5, 5)
	p2 := NewPoint(5, 5)
	if !p1.Equals(p2) {
		t.Fatal("points should be equal")
	}
	if p1.Equals(NewPoint(3, 4)) {
		t.Fatal("points should not be equal")
	}
}

func TestPointHashCode(t *testing.T) {
	p1 := NewPoint(5, 5)
	p2 := NewPoint(5, 5)
	if p1.HashCode() != p2.HashCode() {
		t.Fatal("equal points should have the same hash code")
	}
}

func TestPointString(t *testing.T) {
	if got := NewPoint(3, 4).String(); got != "Point {3, 4}" {
		t.Fatalf("got %q", got)
	}
}

func TestPointOfFloatClone(t *testing.T) {
	pointOfInt := NewPointOfFloat(3, 4)
	cloned := pointOfInt.Impl.Clone()
	clonedPointOfInt, ok := pointImplAsOfFloat(cloned.Impl)
	if !ok {
		t.Fatal("clone did not return a Point.OfFloat")
	}
	if pointOfInt.X != clonedPointOfInt.X || pointOfInt.GetX() != clonedPointOfInt.GetX() {
		t.Fatalf("clone mismatch: %+v vs %+v", pointOfInt, clonedPointOfInt)
	}

	pointOfFloat := NewPointOfFloatXY(3.4, 3.5)
	cf, ok := pointImplAsOfFloat(pointOfFloat.Impl.Clone().Impl)
	if !ok || pointOfFloat.GetX() != cf.GetX() || pointOfFloat.GetY() != cf.GetY() {
		t.Fatalf("float clone mismatch")
	}
}

// --- Rectangle (Test_org_eclipse_swt_graphics_Rectangle) ---

func TestRectangleConstructor(t *testing.T) {
	r := NewRectangle(3, 4, 5, 6)
	if r.X != 3 || r.Y != 4 || r.Width != 5 || r.Height != 6 {
		t.Fatalf("got %+v", r)
	}
}

func TestRectangleAdd(t *testing.T) {
	r1 := NewRectangle(1, 2, 3, 4)
	r2 := NewRectangle(3, 3, 2, 2)
	r1.Add(r2)
	if !r1.Equals(NewRectangle(1, 2, 4, 4)) {
		t.Fatalf("add incorrect: %+v", r1)
	}
	assertPanics(t, func() { NewRectangle(1, 2, 3, 4).Add(nil) })
}

func TestRectangleContains(t *testing.T) {
	r := NewRectangle(1, 2, 3, 4)
	if !r.Contains(1, 2) || !r.Contains(3, 5) {
		t.Fatal("should contain")
	}
	if r.Contains(9, 10) || r.Contains(-1, -1) {
		t.Fatal("should not contain")
	}
	if !r.ContainsPt(NewPoint(3, 4)) {
		t.Fatal("should contain point")
	}
	assertPanics(t, func() { r.ContainsPt(nil) })
}

func TestRectangleEqualsHashCode(t *testing.T) {
	r1 := NewRectangle(5, 4, 3, 2)
	r2 := NewRectangle(5, 4, 3, 2)
	if !r1.Equals(r2) || r1.HashCode() != r2.HashCode() {
		t.Fatal("equal rectangles should compare and hash equal")
	}
	if r1.Equals(NewRectangle(3, 4, 5, 6)) {
		t.Fatal("different rectangles should not be equal")
	}
}

func TestRectangleIntersect(t *testing.T) {
	r1 := NewRectangle(1, 2, 3, 4)
	r1.Intersect(NewRectangle(3, 3, 2, 2))
	if !r1.Equals(NewRectangle(3, 3, 1, 2)) {
		t.Fatalf("intersect incorrect: %+v", r1)
	}
}

func TestRectangleIntersection(t *testing.T) {
	r1 := NewRectangle(1, 2, 3, 4)
	got := r1.Intersection(NewRectangle(3, 3, 2, 2))
	if !got.Equals(NewRectangle(3, 3, 1, 2)) {
		t.Fatalf("intersection incorrect: %+v", got)
	}
}

func TestRectangleIntersects(t *testing.T) {
	r1 := NewRectangle(1, 2, 3, 4)
	r2 := NewRectangle(2, 3, 7, 8)
	if !r1.Intersects(2, 3, 7, 8) || !r1.IntersectsRect(r2) {
		t.Fatal("should intersect")
	}
	if r1.Intersects(200, 300, 400, 500) {
		t.Fatal("should not intersect")
	}
}

func TestRectangleIsEmpty(t *testing.T) {
	if !NewRectangle(1, 2, 0, 0).IsEmpty() {
		t.Fatal("should be empty")
	}
	if NewRectangle(1, 2, 3, 4).IsEmpty() {
		t.Fatal("should not be empty")
	}
}

func TestRectangleString(t *testing.T) {
	if got := NewRectangle(3, 4, 5, 6).String(); got != "Rectangle {3, 4, 5, 6}" {
		t.Fatalf("got %q", got)
	}
}

func TestRectangleUnion(t *testing.T) {
	r1 := NewRectangle(1, 2, 3, 4)
	got := r1.Union(NewRectangle(3, 3, 2, 2))
	if !got.Equals(NewRectangle(1, 2, 4, 4)) {
		t.Fatalf("union incorrect: %+v", got)
	}
}

func TestRectangleOfBothPointsOfFloat(t *testing.T) {
	topLeft := NewPointOfFloatXY(10.5, 20.5)
	dimension := NewPointOfFloatXY(30.5, 40.5)
	rect := RectangleOfTopLeftDimension(&topLeft.Point, &dimension.Point)
	f, ok := rectangleImplAsOfFloat(rect.Impl)
	if !ok {
		t.Fatal("expected a Rectangle.OfFloat")
	}
	if f.GetX() != 10.5 || f.GetY() != 20.5 || f.GetWidth() != 30.5 || f.GetHeight() != 40.5 {
		t.Fatalf("got %+v", f)
	}
}

func TestRectangleOfNeitherPointOfFloat(t *testing.T) {
	rect := RectangleOfTopLeftDimension(NewPoint(1, 2), NewPoint(10, 20))
	if _, ok := rectangleImplAsOfFloat(rect.Impl); ok {
		t.Fatal("expected a plain Rectangle")
	}
	if rect.X != 1 || rect.Y != 2 || rect.Width != 10 || rect.Height != 20 {
		t.Fatalf("got %+v", rect)
	}
}

func TestRectangleOfWithMonitor(t *testing.T) {
	monitor := &Monitor{}
	topLeft := NewPointWithMonitor(10, 20, monitor)
	rect := RectangleOfTopLeftDimension(&topLeft.Point, NewPoint(30, 40))
	wm, ok := rect.Impl.(*Rectangle_WithMonitor)
	if !ok {
		t.Fatal("expected a Rectangle.WithMonitor")
	}
	if wm.GetMonitor() != monitor || wm.GetX() != 10 || wm.GetY() != 20 {
		t.Fatalf("got %+v", wm)
	}
}

// --- RGB (Test_org_eclipse_swt_graphics_RGB) ---

func TestRGBConstructorValidation(t *testing.T) {
	NewRGB(20, 100, 200)
	assertPanics(t, func() { NewRGB(-1, 20, 50) })
	assertPanics(t, func() { NewRGB(256, 20, 50) })
}

func TestRGBEquals(t *testing.T) {
	rgb1 := NewRGB(0, 127, 254)
	rgb2 := NewRGB(0, 127, 254)
	if !rgb1.Equals(rgb2) {
		t.Fatal("same-valued RGBs should be equal")
	}
	if rgb1.Equals(NewRGB(1, 127, 254)) {
		t.Fatal("different RGBs should not be equal")
	}
}

func TestRGBHashCode(t *testing.T) {
	rgb1 := NewRGB(255, 100, 0)
	rgb2 := NewRGB(255, 100, 0)
	if rgb1.HashCode() != rgb2.HashCode() {
		t.Fatal("equal RGBs should hash equal")
	}
}

func TestRGBHSBRoundTrip(t *testing.T) {
	hsb := []float32{0, 0, 0, 0, 1, 1, 220, 0.6, 0.7}
	for i := 0; i < len(hsb); i += 3 {
		rgb1 := NewRGBHueSaturationBrightness(hsb[i], hsb[i+1], hsb[i+2])
		hsb2 := rgb1.GetHSB()
		rgb2 := NewRGBHueSaturationBrightness(hsb2[0], hsb2[1], hsb2[2])
		if !rgb1.Equals(rgb2) {
			t.Fatalf("HSB round trip mismatch for %v: %+v vs %+v", hsb[i:i+3], rgb1, rgb2)
		}
	}
}

func TestRGBString(t *testing.T) {
	if got := NewRGB(0, 100, 200).String(); got != "RGB {0, 100, 200}" {
		t.Fatalf("got %q", got)
	}
}

// --- RGBA (Test_org_eclipse_swt_graphics_RGBA) ---

func TestRGBAConstructorValidation(t *testing.T) {
	NewRGBA(20, 100, 200, 255)
	assertPanics(t, func() { NewRGBA(20, 50, 10, -1) })
	assertPanics(t, func() { NewRGBA(20, 50, 10, 256) })
}

func TestRGBAEqualsHashCode(t *testing.T) {
	rgba1 := NewRGBA(0, 127, 254, 254)
	rgba2 := NewRGBA(0, 127, 254, 254)
	if !rgba1.Equals(rgba2) || rgba1.HashCode() != rgba2.HashCode() {
		t.Fatal("equal RGBAs should compare and hash equal")
	}
	if rgba1.Equals(NewRGBA(1, 127, 254, 254)) {
		t.Fatal("different RGBAs should not be equal")
	}
}

func TestRGBAGetHSBA(t *testing.T) {
	rgba1 := NewRGBAHueSaturationBrightnessAlpha(220, 0.6, 0.7, 0.8)
	hsba := rgba1.GetHSBA()
	rgba2 := NewRGBAHueSaturationBrightnessAlpha(hsba[0], hsba[1], hsba[2], hsba[3])
	if !rgba1.Equals(rgba2) {
		t.Fatalf("HSBA round trip mismatch: %+v vs %+v", rgba1, rgba2)
	}
}

func TestRGBAString(t *testing.T) {
	if got := NewRGBA(0, 100, 200, 255).String(); got != "RGBA {0, 100, 200, 255}" {
		t.Fatalf("got %q", got)
	}
}
