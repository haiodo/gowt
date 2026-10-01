package main

import (
	"fmt"
	"strconv"
	"strings"
)

// LField is one laid-out C field after flattening nested by-value structs (Java-style names
// "analysis_level"). Synthetic fields are storage units of bitfields and have no C name.
type LField struct {
	Name, Go  string
	CPath     string // C member path (analysis.level) for offsetof
	Off, Size int
	Synthetic bool
}

type BitField struct {
	Name string
	Unit string // Go name of the storage unit field
	Pos  int
}

type Layout struct {
	CType  string
	Fields []LField
	Bits   []BitField
	Size   int
	Align  int
	Union  bool
}

func goSize(t string) (size, align int) {
	switch t {
	case "int8", "uint8":
		return 1, 1
	case "int16", "uint16":
		return 2, 2
	case "int32", "uint32", "float32":
		return 4, 4
	}
	return 8, 8
}

func alignUp(n, a int) int { return (n + a - 1) / a * a }

// fieldInfo returns size, alignment and Go type of a field's C type; a by-value struct comes back
// as ("", record) so the caller can flatten it.
func (g *gen) fieldInfo(ix *Index, ns string, f *Field) (size, align int, goType string, nested *Layout) {
	ct := ""
	if f.Type != nil {
		ct = f.Type.CType
	}
	if f.Callback != nil {
		return 8, 8, "int64", nil
	}
	if f.Array != nil {
		n, _ := strconv.Atoi(f.Array.FixedSize)
		if f.Array.FixedSize == "" || f.Array.Type == nil {
			return 8, 8, "int64", nil
		}
		sub := Field{}
		sub.Type = f.Array.Type
		es, ea, et, nl := g.fieldInfo(ix, ns, &sub)
		if nl != nil {
			et = exportName(nl.CType)
		}
		return es * n, ea, fmt.Sprintf("[%d]%s", n, et), nil
	}
	if ct == "" && f.Type != nil {
		ct = f.Type.Name
		if c, ok := ix.ByName[ct]; ok {
			ct = c
		} else if c, ok := ix.ByName[ns+"."+ct]; ok {
			ct = c
		}
	}
	base, stars := normCType(ct)
	if stars > 0 || ix.Cbs[base] {
		return 8, 8, "int64", nil
	}
	if p := ix.primOf(base); p != "" {
		switch p {
		case "bool":
			return 4, 4, "int32", nil
		case "int8":
			return 1, 1, "int8", nil
		case "int16":
			return 2, 2, "int16", nil
		}
		s, a := goSize(p)
		return s, a, p, nil
	}
	if rec := ix.structOf(base); rec != nil {
		l := g.layoutOf(ix, rec)
		return l.Size, l.Align, exportName(base), l
	}
	if f.Type != nil && f.Type.Name == "gpointer" {
		return 8, 8, "int64", nil
	}
	g.warn("field %s: unknown type %q", f.Name, ct)
	return 8, 8, "int64", nil
}

func (g *gen) layoutOf(ix *Index, r *Record) *Layout {
	if l, ok := g.layouts[r.CType]; ok && r.CType != "" {
		return l
	}
	l := &Layout{CType: r.CType, Align: 1, Union: r.IsUnion}
	if r.CType != "" {
		g.layouts[r.CType] = l
	}
	ns := strings.SplitN(ix.NSOf[r.CType], "-", 2)[0]
	off := 0
	unit, used, unitSize := "", 0, 0
	for i := range r.Fields {
		f := &r.Fields[i]
		if f.Record != nil || f.Union != nil {
			g.warn("%s.%s: anonymous nested aggregate", r.CType, f.Name)
			continue
		}
		size, align, gt, nested := g.fieldInfo(ix, ns, f)
		if f.Bits != "" {
			n, _ := strconv.Atoi(f.Bits)
			if unit == "" || used+n > unitSize*8 || size != unitSize {
				off = alignUp(off, align)
				unit, used, unitSize = fmt.Sprintf("bits%d", off), 0, size
				l.Fields = append(l.Fields, LField{Name: unit, CPath: unit, Go: gt, Off: off, Size: size, Synthetic: true})
				off += size
				l.Align = max(l.Align, align)
			}
			if n == 1 {
				l.Bits = append(l.Bits, BitField{Name: f.Name, Unit: unit, Pos: used})
			}
			used += n
			continue
		}
		unit = ""
		at := off
		if r.IsUnion {
			at = 0
		} else {
			at = alignUp(off, align)
		}
		if nested != nil {
			prefix := f.Name + "_"
			if f.Name == "attr" || f.Name == "parent_instance" || f.Name == "parent_class" {
				prefix = ""
			}
			for _, sf := range nested.Fields {
				cp := f.Name + "." + sf.CPath
				if prefix == "" {
					cp = f.Name + "." + sf.CPath
				}
				sf.Name, sf.CPath, sf.Off = prefix+sf.Name, cp, sf.Off+at
				l.Fields = append(l.Fields, sf)
			}
			for _, b := range nested.Bits {
				b.Name, b.Unit = prefix+b.Name, prefix+b.Unit
				l.Bits = append(l.Bits, b)
			}
		} else {
			l.Fields = append(l.Fields, LField{Name: f.Name, CPath: f.Name, Go: gt, Off: at, Size: size})
		}
		l.Align = max(l.Align, align)
		if r.IsUnion {
			l.Size = max(l.Size, size)
		} else {
			off = at + size
		}
	}
	if r.IsUnion {
		l.Size = alignUp(l.Size, l.Align)
	} else {
		l.Size = alignUp(off, l.Align)
	}
	return l
}

func exportName(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// emitStruct pads where Go's layout would drift from C's; a union keeps its first member. Bitfields
// and wide structs get extra Go fields that unpack() fills after C or memmove wrote the bytes.
func (g *gen) emitStruct(w *strings.Builder, name string, l *Layout, extra []GoParam, wide, bitBool bool) {
	fmt.Fprintf(w, "type %s struct {\n", name)
	goOff := 0
	if wide {
		fmt.Fprintf(w, "\traw [%d]byte\n", l.Size)
	}
	for _, f := range l.Fields {
		if l.Union && goOff > 0 {
			break
		}
		if wide {
			if !f.Synthetic {
				fmt.Fprintf(w, "\t%s int32\n", exportName(f.Name))
			}
			continue
		}
		_, a := goSize(f.Go)
		if !isPrimGo(f.Go) {
			a = g.alignOfGo(f.Go)
		}
		goOff = alignUp(goOff, a)
		if f.Off > goOff {
			fmt.Fprintf(w, "\t_ [%d]byte\n", f.Off-goOff)
			goOff = f.Off
		}
		fn := exportName(f.Name)
		if strings.HasPrefix(fn, "_") { // reserved slots repeat across flattened parents
			fn = "_"
		}
		fmt.Fprintf(w, "\t%s %s\n", fn, f.Go)
		goOff += f.Size
	}
	if !wide && l.Size > goOff {
		fmt.Fprintf(w, "\t_ [%d]byte\n", l.Size-goOff)
	}
	bt := "int32"
	if bitBool {
		bt = "bool"
	}
	for _, b := range l.Bits {
		fmt.Fprintf(w, "\t%s %s\n", exportName(b.Name), bt)
	}
	for _, e := range extra {
		fmt.Fprintf(w, "\t%s %s\n", e.Name, e.Type)
	}
	w.WriteString("}\n\n")
	if !wide && len(l.Bits) == 0 {
		return
	}
	fmt.Fprintf(w, "func (s *%s) unpack() {\n", name)
	if wide {
		for _, f := range l.Fields {
			if !f.Synthetic {
				fmt.Fprintf(w, "\ts.%s = int32(*(*%s)(unsafe.Pointer(&s.raw[%d])))\n", exportName(f.Name), f.Go, f.Off)
			}
		}
	}
	for _, b := range l.Bits {
		if bitBool {
			fmt.Fprintf(w, "\ts.%s = s.%s&(1<<%d) != 0\n", exportName(b.Name), exportName(b.Unit), b.Pos)
		} else {
			fmt.Fprintf(w, "\ts.%s = s.%s>>%d&1\n", exportName(b.Name), exportName(b.Unit), b.Pos)
		}
	}
	w.WriteString("}\n\n")
}

func isPrimGo(t string) bool {
	switch t {
	case "int8", "int16", "int32", "int64", "float32", "float64":
		return true
	}
	return false
}

func (g *gen) alignOfGo(t string) int {
	if i := strings.Index(t, "]"); strings.HasPrefix(t, "[") && i > 0 {
		return g.alignOfGo(t[i+1:])
	}
	for ct, l := range g.layouts {
		if exportName(ct) == t {
			return l.Align
		}
	}
	_, a := goSize(t)
	return a
}
