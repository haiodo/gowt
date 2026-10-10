package swt

// Comparator.comparingInt(StyledText::getX).thenComparingInt(p -> p.y): selection ranges by start, then end.
var StyledTextSELECTION_COMPARATOR = func(a, b *Point) int32 {
	if a.X != b.X {
		if a.X < b.X {
			return -1
		}
		return 1
	}
	if a.Y < b.Y {
		return -1
	}
	if a.Y > b.Y {
		return 1
	}
	return 0
}
