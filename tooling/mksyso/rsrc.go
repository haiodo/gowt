package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode/utf16"
)

// Resource types (winuser.h).
const (
	rtIcon      = 3
	rtGroupIcon = 14
	rtVersion   = 16
	rtManifest  = 24
	langEnUS    = 0x0409
)

type resource struct {
	typ, id uint16
	data    []byte
}

var le = binary.LittleEndian

// coff returns a COFF object with one .rsrc section holding the resources in a three-level tree
// (type, name, language), for go build to link into a Windows executable.
func coff(arch string, rs []resource) ([]byte, error) {
	var machine, addr32nb uint16
	switch arch {
	case "amd64":
		machine, addr32nb = 0x8664, 3 // IMAGE_REL_AMD64_ADDR32NB
	case "arm64":
		machine, addr32nb = 0xAA64, 2 // IMAGE_REL_ARM64_ADDR32NB
	default:
		return nil, fmt.Errorf("unknown -arch %q", arch)
	}
	if len(rs) == 0 {
		return nil, errors.New("nothing to write: give -manifest, -icon or -version")
	}
	sort.Slice(rs, func(i, j int) bool {
		if rs[i].typ != rs[j].typ {
			return rs[i].typ < rs[j].typ
		}
		return rs[i].id < rs[j].id
	})
	var types []uint16
	names := map[uint16][]int{} // type -> indices into rs
	for i, r := range rs {
		if len(names[r.typ]) == 0 {
			types = append(types, r.typ)
		}
		names[r.typ] = append(names[r.typ], i)
	}

	// directory layout: root, then per type its directory and the directories of its names
	dirSize := func(n int) int { return 16 + 8*n }
	off := dirSize(len(types))
	typeOff := map[uint16]int{}
	nameOff := make([]int, len(rs))
	for _, t := range types {
		typeOff[t] = off
		off += dirSize(len(names[t]))
	}
	for _, t := range types {
		for _, i := range names[t] {
			nameOff[i] = off
			off += dirSize(1)
		}
	}
	entryOff := make([]int, len(rs))
	for i := range rs {
		entryOff[i] = off
		off += 16
	}

	var rsrc bytes.Buffer
	w := func(v ...any) {
		for _, x := range v {
			binary.Write(&rsrc, le, x)
		}
	}
	dir := func(n int) { w(uint32(0), uint32(0), uint32(0), uint16(0), uint16(n)) }
	const sub = 0x80000000
	dir(len(types))
	for _, t := range types {
		w(uint32(t), uint32(sub|typeOff[t]))
	}
	for _, t := range types {
		dir(len(names[t]))
		for _, i := range names[t] {
			w(uint32(rs[i].id), uint32(sub|nameOff[i]))
		}
	}
	for _, t := range types {
		for _, i := range names[t] {
			dir(1)
			w(uint32(langEnUS), uint32(entryOff[i]))
		}
	}
	dataOff := off
	datas := make([]int, len(rs))
	for i, r := range rs {
		datas[i] = dataOff
		dataOff += (len(r.data) + 3) &^ 3
	}
	for i, r := range rs {
		w(uint32(datas[i]), uint32(len(r.data)), uint32(0), uint32(0))
	}
	for _, r := range rs {
		rsrc.Write(r.data)
		for rsrc.Len()%4 != 0 {
			rsrc.WriteByte(0)
		}
	}

	const headers = 20 + 40
	relocOff := headers + rsrc.Len()
	symOff := relocOff + 10*len(rs)
	var f bytes.Buffer
	fw := func(v ...any) {
		for _, x := range v {
			binary.Write(&f, le, x)
		}
	}
	fw(machine, uint16(1), uint32(0), uint32(symOff), uint32(1), uint16(0), uint16(0)) // file header
	f.WriteString(".rsrc\x00\x00\x00")
	fw(uint32(0), uint32(0), uint32(rsrc.Len()), uint32(headers), uint32(relocOff), uint32(0), uint16(len(rs)), uint16(0), uint32(0x40000040))
	f.Write(rsrc.Bytes())
	for i := range rs {
		fw(uint32(entryOff[i]), uint32(0), addr32nb) // the data entry's OffsetToData, against symbol 0
	}
	f.WriteString(".rsrc\x00\x00\x00")
	fw(uint32(0), uint16(1), uint16(0), uint8(3), uint8(0)) // static section symbol
	fw(uint32(4))                                           // empty string table
	return f.Bytes(), nil
}

// icons turns the -icon argument, an .ico file or PNG files separated by commas, into the RT_ICON
// resources and the RT_GROUP_ICON that lists them. PNG images are stored as they are, which
// Windows has read since Vista.
func icons(arg string) ([]resource, error) {
	type image struct {
		w, h int
		data []byte
	}
	var imgs []image
	for _, p := range strings.Split(arg, ",") {
		p = strings.TrimSpace(p)
		b, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}
		if strings.EqualFold(filepath.Ext(p), ".ico") {
			if len(b) < 6 || le.Uint16(b[2:]) != 1 {
				return nil, fmt.Errorf("%s is not an .ico file", p)
			}
			for i := 0; i < int(le.Uint16(b[4:])); i++ {
				e := b[6+16*i:]
				w, h := int(e[0]), int(e[1])
				size, off := int(le.Uint32(e[8:])), int(le.Uint32(e[12:]))
				if off+size > len(b) {
					return nil, fmt.Errorf("%s: image %d is cut off", p, i)
				}
				imgs = append(imgs, image{w, h, b[off : off+size]})
			}
			continue
		}
		cfg, err := png.DecodeConfig(bytes.NewReader(b))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", p, err)
		}
		if cfg.Width != cfg.Height || cfg.Width > 256 {
			return nil, fmt.Errorf("%s is %dx%d: an icon image is square and at most 256", p, cfg.Width, cfg.Height)
		}
		imgs = append(imgs, image{cfg.Width, cfg.Height, b})
	}
	if len(imgs) == 0 {
		return nil, errors.New("-icon names no image")
	}
	var rs []resource
	group := []byte{0, 0, 1, 0, byte(len(imgs)), byte(len(imgs) >> 8)}
	for i, im := range imgs {
		id := uint16(i + 1)
		rs = append(rs, resource{rtIcon, id, im.data})
		e := make([]byte, 14)
		e[0], e[1] = byte(im.w), byte(im.h) // 256 is 0
		le.PutUint16(e[4:], 1)              // planes
		le.PutUint16(e[6:], 32)             // bits per pixel
		le.PutUint32(e[8:], uint32(len(im.data)))
		le.PutUint16(e[12:], id)
		group = append(group, e...)
	}
	return append(rs, resource{rtGroupIcon, 1, group}), nil
}

type versionInfo struct {
	version                                                      string // 1.2.3.4, fewer parts are zero-filled
	product, company, description, copyright, original, internal string
}

// parseVersion reads up to four dot-separated numbers.
func parseVersion(s string) (v [4]uint16, err error) {
	parts := strings.Split(s, ".")
	if len(parts) > 4 {
		return v, fmt.Errorf("version %q has more than four parts", s)
	}
	for i, p := range parts {
		n, err := strconv.ParseUint(p, 10, 16)
		if err != nil {
			return v, fmt.Errorf("version %q: %w", s, err)
		}
		v[i] = uint16(n)
	}
	return v, nil
}

func wstr(s string) []byte {
	var b []byte
	for _, u := range utf16.Encode([]rune(s)) {
		b = le.AppendUint16(b, u)
	}
	return le.AppendUint16(b, 0)
}

func pad4(b []byte) []byte {
	for len(b)%4 != 0 {
		b = append(b, 0)
	}
	return b
}

// node is one VS_VERSIONINFO structure: length, value length, type, key, value and children.
func node(key string, valueLen int, text bool, value []byte, children ...[]byte) []byte {
	b := make([]byte, 6)
	if text {
		le.PutUint16(b[4:], 1)
	}
	le.PutUint16(b[2:], uint16(valueLen))
	b = pad4(append(b, wstr(key)...))
	b = append(b, value...)
	for _, c := range children {
		b = append(pad4(b), c...)
	}
	le.PutUint16(b, uint16(len(b)))
	return b
}

// versionResource is the RT_VERSION data: fixed numbers plus the string table of en-US, Unicode.
func versionResource(v versionInfo) ([]byte, error) {
	num := [4]uint16{}
	if v.version != "" {
		var err error
		if num, err = parseVersion(v.version); err != nil {
			return nil, err
		}
	}
	ver := fmt.Sprintf("%d.%d.%d.%d", num[0], num[1], num[2], num[3])
	fixed := make([]byte, 52)
	for i, x := range []uint32{0xFEEF04BD, 0x00010000, uint32(num[0])<<16 | uint32(num[1]), uint32(num[2])<<16 | uint32(num[3]),
		uint32(num[0])<<16 | uint32(num[1]), uint32(num[2])<<16 | uint32(num[3]), 0x3F, 0, 0x40004, 1, 0, 0, 0} {
		le.PutUint32(fixed[4*i:], x)
	}
	var strs [][]byte
	add := func(k, val string) {
		if val != "" {
			strs = append(strs, pad4(node(k, len(utf16.Encode([]rune(val)))+1, true, wstr(val))))
		}
	}
	add("CompanyName", v.company)
	add("FileDescription", v.description)
	add("FileVersion", ver)
	add("InternalName", v.internal)
	add("LegalCopyright", v.copyright)
	add("OriginalFilename", v.original)
	add("ProductName", v.product)
	add("ProductVersion", ver)
	table := node("040904B0", 0, true, nil, strs...)
	translation := node("Translation", 4, false, le.AppendUint32(nil, 0x04B00409))
	return node("VS_VERSION_INFO", 52, false, fixed,
		node("StringFileInfo", 0, true, nil, table),
		node("VarFileInfo", 0, true, nil, translation)), nil
}
