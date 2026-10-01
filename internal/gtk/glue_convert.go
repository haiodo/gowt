//go:build linux

package gtk

import "unicode/utf16"

// SWT's Converter: Java strings and char[] to NUL-terminated UTF-8 byte arrays and back.

func utf8z(s string, terminate bool) []int8 {
	n := len(s)
	if terminate {
		n++
	}
	b := make([]int8, n)
	for i := 0; i < len(s); i++ {
		b[i] = int8(s[i])
	}
	return b
}

func ConverterWcsToMbcs(s string, terminate bool) []int8 { return utf8z(s, terminate) }

func ConverterJavaStringToCString(s string) []int8 { return utf8z(s, true) }

func ConverterWcsToMbcsCharsTerminate(chars []uint16, terminate bool) []int8 {
	return utf8z(string(utf16.Decode(chars)), terminate)
}

// ConverterWcsToMbcsCh converts one character; keysyms below 0x100 and Unicode values are their own code.
func ConverterWcsToMbcsCh(ch uint16) uint16 { return ch }

func ConverterMbcsToWcs(b []int8) []uint16 {
	raw := make([]byte, len(b))
	for i, c := range b {
		raw[i] = byte(c)
	}
	return utf16.Encode([]rune(string(raw)))
}

// ConverterCCharPtrToJavaString reads a NUL-terminated C string and optionally g_free()s it.
func ConverterCCharPtrToJavaString(p int64, free bool) string {
	if p == 0 {
		return ""
	}
	s := string(goBytes(p, -1))
	if free {
		OSG_free(p)
	}
	return s
}
