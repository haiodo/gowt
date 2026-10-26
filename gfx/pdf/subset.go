package pdf

import (
	"bytes"
	"encoding/binary"
	"sort"
)

// subsetTTF returns a TrueType file that keeps the outlines of the used glyphs and of the
// components they are built from. Glyph numbers stay as they are, so the text strings need no
// remapping; the other glyphs become empty and the layout, cmap and name tables go, which is
// most of a large font. It returns nil for a file it cannot read, and the caller embeds the whole.
func subsetTTF(data []byte, used []uint16) []byte {
	if len(data) < 12 {
		return nil
	}
	tables := readTables(data)
	if tables == nil {
		return nil
	}
	head, maxp, loca, glyf := tables["head"], tables["maxp"], tables["loca"], tables["glyf"]
	if len(head) < 54 || len(maxp) < 6 || loca == nil || glyf == nil {
		return nil
	}
	numGlyphs := int(binary.BigEndian.Uint16(maxp[4:]))
	long := binary.BigEndian.Uint16(head[50:]) != 0
	offsetOf := func(g int) (int, bool) {
		if long {
			if 4*g+4 > len(loca) {
				return 0, false
			}
			return int(binary.BigEndian.Uint32(loca[4*g:])), true
		}
		if 2*g+2 > len(loca) {
			return 0, false
		}
		return 2 * int(binary.BigEndian.Uint16(loca[2*g:])), true
	}
	body := func(g int) []byte {
		a, ok1 := offsetOf(g)
		b, ok2 := offsetOf(g + 1)
		if !ok1 || !ok2 || a > b || b > len(glyf) {
			return nil
		}
		return glyf[a:b]
	}
	keep := map[int]bool{0: true}
	var walk func(g int)
	walk = func(g int) {
		if g < 0 || g >= numGlyphs || keep[g] && g != 0 {
			return
		}
		keep[g] = true
		b := body(g)
		if len(b) < 10 || int16(binary.BigEndian.Uint16(b)) >= 0 {
			return
		}
		for p := 10; p+4 <= len(b); {
			flags := binary.BigEndian.Uint16(b[p:])
			walk(int(binary.BigEndian.Uint16(b[p+2:])))
			p += 4
			if flags&1 != 0 {
				p += 4
			} else {
				p += 2
			}
			switch {
			case flags&0x08 != 0:
				p += 2
			case flags&0x40 != 0:
				p += 4
			case flags&0x80 != 0:
				p += 8
			}
			if flags&0x20 == 0 {
				break
			}
		}
	}
	walk(0)
	for _, g := range used {
		walk(int(g))
	}
	var newGlyf bytes.Buffer
	newLoca := make([]byte, 4*(numGlyphs+1))
	for g := 0; g < numGlyphs; g++ {
		binary.BigEndian.PutUint32(newLoca[4*g:], uint32(newGlyf.Len()))
		if keep[g] {
			newGlyf.Write(body(g))
			for newGlyf.Len()%4 != 0 {
				newGlyf.WriteByte(0)
			}
		}
	}
	binary.BigEndian.PutUint32(newLoca[4*numGlyphs:], uint32(newGlyf.Len()))
	newHead := append([]byte(nil), head...)
	binary.BigEndian.PutUint16(newHead[50:], 1)
	binary.BigEndian.PutUint32(newHead[8:], 0)

	out := map[string][]byte{"head": newHead, "glyf": newGlyf.Bytes(), "loca": newLoca}
	for _, t := range []string{"hhea", "maxp", "hmtx", "cvt ", "fpgm", "prep", "OS/2", "post"} {
		if b, ok := tables[t]; ok {
			out[t] = b
		}
	}
	if post, ok := out["post"]; ok && len(post) >= 4 { // version 3: no glyph names
		p := append([]byte(nil), post[:min(len(post), 32)]...)
		binary.BigEndian.PutUint32(p, 0x00030000)
		out["post"] = p
	}
	return buildTTF(out)
}

// readTables returns the tables of an sfnt file by tag, or nil when the directory is damaged.
func readTables(data []byte) map[string][]byte {
	n := int(binary.BigEndian.Uint16(data[4:]))
	tables := map[string][]byte{}
	for i := 0; i < n; i++ {
		rec := 12 + 16*i
		if rec+16 > len(data) {
			return nil
		}
		off, ln := int(binary.BigEndian.Uint32(data[rec+8:])), int(binary.BigEndian.Uint32(data[rec+12:]))
		if off < 0 || ln < 0 || off+ln > len(data) {
			return nil
		}
		tables[string(data[rec:rec+4])] = data[off : off+ln]
	}
	return tables
}

// buildTTF writes a TrueType file of the tables.
func buildTTF(out map[string][]byte) []byte {
	tags := make([]string, 0, len(out))
	for t := range out {
		tags = append(tags, t)
	}
	sort.Strings(tags)
	var f bytes.Buffer
	f.Write([]byte{0, 1, 0, 0})
	k := len(tags)
	pow, lg := 1, 0
	for pow*2 <= k {
		pow *= 2
		lg++
	}
	binary.Write(&f, binary.BigEndian, [4]uint16{uint16(k), uint16(pow * 16), uint16(lg), uint16(k*16 - pow*16)})
	off := 12 + 16*k
	for _, t := range tags {
		b := out[t]
		f.WriteString(t)
		binary.Write(&f, binary.BigEndian, [3]uint32{checksum(b), uint32(off), uint32(len(b))})
		off += (len(b) + 3) &^ 3
	}
	for _, t := range tags {
		f.Write(out[t])
		for f.Len()%4 != 0 {
			f.WriteByte(0)
		}
	}
	return f.Bytes()
}

func checksum(b []byte) uint32 {
	var sum uint32
	for i := 0; i < len(b); i += 4 {
		var w [4]byte
		copy(w[:], b[i:])
		sum += binary.BigEndian.Uint32(w[:])
	}
	return sum
}
