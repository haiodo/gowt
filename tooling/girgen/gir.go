package main

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const cNS = "http://www.gtk.org/introspection/c/1.0"

// GIR element subset: only what the generator reads.
type Repo struct {
	NS struct {
		Name      string     `xml:"name,attr"`
		Version   string     `xml:"version,attr"`
		Funcs     []Callable `xml:"function"`
		Consts    []Const    `xml:"constant"`
		Enums     []Enum     `xml:"enumeration"`
		Flags     []Enum     `xml:"bitfield"`
		Records   []Record   `xml:"record"`
		Unions    []Record   `xml:"union"`
		Classes   []Record   `xml:"class"`
		Ifaces    []Record   `xml:"interface"`
		Aliases   []Alias    `xml:"alias"`
		Callbacks []Callable `xml:"callback"`
	} `xml:"namespace"`
}

type Type struct {
	Name  string `xml:"name,attr"`
	CType string `xml:"http://www.gtk.org/introspection/c/1.0 type,attr"`
}

type TypeOrArray struct {
	Type    *Type     `xml:"type"`
	Array   *Array    `xml:"array"`
	Varargs *struct{} `xml:"varargs"`
}

type Array struct {
	CType     string `xml:"http://www.gtk.org/introspection/c/1.0 type,attr"`
	FixedSize string `xml:"fixed-size,attr"`
	Type      *Type  `xml:"type"`
}

type Param struct {
	Name        string `xml:"name,attr"`
	Direction   string `xml:"direction,attr"`
	CallerAlloc string `xml:"caller-allocates,attr"`
	TypeOrArray
}

type Callable struct {
	Name   string                `xml:"name,attr"`
	CIdent string                `xml:"http://www.gtk.org/introspection/c/1.0 identifier,attr"`
	CType  string                `xml:"http://www.gtk.org/introspection/c/1.0 type,attr"`
	Throws string                `xml:"throws,attr"`
	Ret    struct{ TypeOrArray } `xml:"return-value"`
	Inst   *Param                `xml:"parameters>instance-parameter"`
	Params []Param               `xml:"parameters>parameter"`
}

type Const struct {
	Name   string `xml:"name,attr"`
	Value  string `xml:"value,attr"`
	CType  string `xml:"http://www.gtk.org/introspection/c/1.0 type,attr"`
	CIdent string `xml:"http://www.gtk.org/introspection/c/1.0 identifier,attr"`
	Type   *Type  `xml:"type"`
}

type Member struct {
	Name   string `xml:"name,attr"`
	Value  string `xml:"value,attr"`
	CIdent string `xml:"http://www.gtk.org/introspection/c/1.0 identifier,attr"`
}

type Enum struct {
	Name    string   `xml:"name,attr"`
	CType   string   `xml:"http://www.gtk.org/introspection/c/1.0 type,attr"`
	Members []Member `xml:"member"`
}

type Field struct {
	Name    string `xml:"name,attr"`
	Bits    string `xml:"bits,attr"`
	Private string `xml:"private,attr"`
	TypeOrArray
	Callback *Callable `xml:"callback"`
	// Anonymous nested record/union (e.g. inside a union).
	Record *Record `xml:"record"`
	Union  *Record `xml:"union"`
}

type Record struct {
	Name      string     `xml:"name,attr"`
	CType     string     `xml:"http://www.gtk.org/introspection/c/1.0 type,attr"`
	Disguised string     `xml:"disguised,attr"`
	Fields    []Field    `xml:"field"`
	Methods   []Callable `xml:"method"`
	Ctors     []Callable `xml:"constructor"`
	Funcs     []Callable `xml:"function"`
	Props     []struct {
		Name string `xml:"name,attr"`
	} `xml:"property"`
	Signals []struct {
		Name string `xml:"name,attr"`
	} `xml:"signal"`
	// Union members: nested records are the alternatives.
	Records []Record `xml:"record"`
	IsUnion bool     `xml:"-"`
}

type Alias struct {
	Name  string `xml:"name,attr"`
	CType string `xml:"http://www.gtk.org/introspection/c/1.0 type,attr"`
	Type  *Type  `xml:"type"`
}

// Index of everything callable or nameable by a C identifier across the loaded namespaces.
type Index struct {
	Funcs     map[string]*Callable // c:identifier
	FuncNS    map[string]string    // c:identifier -> namespace-version ("Gtk-3.0")
	Consts    map[string]string    // C name -> value
	Types     map[string]*Record   // c:type -> record/class/union
	Enums     map[string]bool      // c:type of enums/bitfields
	Aliases   map[string]string    // c:type -> target c:type or primitive
	Cbs       map[string]bool      // callback c:type
	Props     map[string]bool      // GObject property/signal/style names (hyphenated)
	NSOf      map[string]string    // c:type -> namespace file
	ByName    map[string]string    // "Gdk.EventAny" -> c:type, for fields that carry no c:type
	StrConsts map[string]bool      // constants of type utf8
}

func loadRepo(dir, file string) (*Repo, error) {
	b, err := os.ReadFile(filepath.Join(dir, file+".gir"))
	if err != nil {
		return nil, err
	}
	var r Repo
	if err := xml.Unmarshal(b, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

func newIndex() *Index {
	return &Index{Funcs: map[string]*Callable{}, FuncNS: map[string]string{}, Consts: map[string]string{},
		Types: map[string]*Record{}, Enums: map[string]bool{}, Aliases: map[string]string{},
		Cbs: map[string]bool{}, Props: map[string]bool{}, NSOf: map[string]string{}, ByName: map[string]string{}, StrConsts: map[string]bool{}}
}

func (ix *Index) addFn(c *Callable, ns string) {
	if c.CIdent == "" {
		return
	}
	if _, dup := ix.Funcs[c.CIdent]; !dup {
		ix.Funcs[c.CIdent] = c
		ix.FuncNS[c.CIdent] = ns
	}
}

func (ix *Index) addRecord(r *Record, ns string) {
	if r.Name != "" && r.CType != "" {
		ix.ByName[strings.SplitN(ns, "-", 2)[0]+"."+r.Name] = r.CType
	}
	if r.CType != "" {
		if _, dup := ix.Types[r.CType]; !dup {
			ix.Types[r.CType] = r
			ix.NSOf[r.CType] = ns
		}
	}
	for i := range r.Methods {
		ix.addFn(&r.Methods[i], ns)
	}
	for i := range r.Ctors {
		ix.addFn(&r.Ctors[i], ns)
	}
	for i := range r.Funcs {
		ix.addFn(&r.Funcs[i], ns)
	}
	for _, p := range r.Props {
		ix.Props[p.Name] = true
	}
	for _, s := range r.Signals {
		ix.Props[s.Name] = true
	}
}

// Adds a repository; earlier repositories win on duplicate identifiers.
func (ix *Index) add(r *Repo, tag string) {
	n := &r.NS
	for i := range n.Funcs {
		ix.addFn(&n.Funcs[i], tag)
	}
	for _, c := range n.Consts {
		name := c.CType
		if name == "" {
			name = c.CIdent
		}
		if _, dup := ix.Consts[name]; !dup && name != "" {
			ix.Consts[name] = c.Value
			ix.StrConsts[name] = c.Type != nil && c.Type.Name == "utf8"
		}
	}
	for _, list := range [][]Enum{n.Enums, n.Flags} {
		for _, e := range list {
			if e.CType != "" {
				ix.Enums[e.CType] = true
				ix.ByName[strings.SplitN(tag, "-", 2)[0]+"."+e.Name] = e.CType
			}
			for _, m := range e.Members {
				if _, dup := ix.Consts[m.CIdent]; !dup && m.CIdent != "" {
					ix.Consts[m.CIdent] = m.Value
				}
			}
		}
	}
	for i := range n.Records {
		ix.addRecord(&n.Records[i], tag)
	}
	for i := range n.Unions {
		n.Unions[i].IsUnion = true
		ix.addRecord(&n.Unions[i], tag)
	}
	for i := range n.Classes {
		ix.addRecord(&n.Classes[i], tag)
	}
	for i := range n.Ifaces {
		ix.addRecord(&n.Ifaces[i], tag)
	}
	for _, a := range n.Aliases {
		t := ""
		if a.Type != nil {
			t = a.Type.CType
			if t == "" {
				t = a.Type.Name
			}
		}
		if _, dup := ix.Aliases[a.CType]; !dup {
			ix.Aliases[a.CType] = t
		}
	}
	for _, c := range n.Callbacks {
		ix.Cbs[c.CType] = true
	}
}

func parseInt(s string) (int64, bool) {
	s = strings.TrimSpace(s)
	if v, err := strconv.ParseInt(s, 0, 64); err == nil {
		return v, true
	}
	if v, err := strconv.ParseUint(s, 0, 64); err == nil {
		return int64(v), true
	}
	return 0, false
}
