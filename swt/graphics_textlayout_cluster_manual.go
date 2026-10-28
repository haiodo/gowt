package swt

import "github.com/haiodo/gowt/internal/jrt"

// clusterJoined: a Uniscribe cluster stop at offset is inside an emoji sequence; Uniscribe stops
// at each code point of a ZWJ family. Ceiling: the emoji rules of UAX #29, not its full algorithm.
func clusterJoined(s string, offset int32) bool {
	n := jrt.StringLength(s)
	if offset <= 0 || offset >= n {
		return false
	}
	if jrt.CharAt(s, offset-1) == 0x200D {
		return true
	}
	switch c := jrt.CharAt(s, offset); {
	case c == 0x200D, 0xFE00 <= c && c <= 0xFE0F:
		return true
	case offset+1 < n && c == 0xD83C:
		lo := jrt.CharAt(s, offset+1)
		return 0xDFFB <= lo && lo <= 0xDFFF // U+1F3FB..U+1F3FF
	case offset+1 < n && c == 0xDB40:
		lo := jrt.CharAt(s, offset+1)
		return 0xDC20 <= lo && lo <= 0xDC7F // U+E0020..U+E007F
	}
	return false
}
