package jrt

import "net/url"

// The java.net subset EchoHttpServer (tests/swttests) uses.

// Charset stands for java.nio.charset.Charset: everything is UTF-8 here.
type Charset struct{}

var StandardCharsetsUTF_8 = &Charset{}

func URLEncoderEncode(s string, _ *Charset) string { return url.QueryEscape(s) }

func URLDecoderDecode(s string, _ *Charset) string {
	d, err := url.QueryUnescape(s)
	if err != nil {
		return s
	}
	return d
}

// GetQuery is URI.getQuery(): the raw-decoded query, null (the empty string here) without one.
func (u *URI) GetQuery() string {
	p, err := url.Parse(u.s)
	if err != nil {
		return ""
	}
	q, err := url.PathUnescape(p.RawQuery)
	if err != nil {
		return p.RawQuery
	}
	return q
}

// GetBytes is String.getBytes(): the UTF-8 bytes as Java's signed bytes.
func GetBytes(s string) []int8 {
	b := []byte(s)
	out := make([]int8, len(b))
	for i, v := range b {
		out[i] = int8(v)
	}
	return out
}

// WriteBytes is OutputStream.write(byte[]).
func WriteBytes(o OutputStream, b []int8) { o.WriteRange(b, 0, int32(len(b))) }

// StringFromBytes is new String(byte[], UTF_8).
func StringFromBytes(b []int8) string { return string(toBytes(b)) }
