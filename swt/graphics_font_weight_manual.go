package swt

// ClampFontWeight limits a CSS font weight to 1..1000.
func ClampFontWeight(w int32) int32 { return min(max(w, 1), 1000) }
