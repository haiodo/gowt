// Package genplan packs generated declaration chunks into deterministic, meaningfully
// named part files. It is the shared planner behind the GIR generators (girgen, webkitgen)
// and mirrors the j2go FilePlanner contract: chunks group by name family, consecutive
// groups fill a part up to the target line count, a part never exceeds the ceiling except
// for a single indivisible chunk, and part names carry the sanitized group name, never an
// opaque part number.
package genplan

import (
	"fmt"
	"sort"
	"strings"
)

// Chunk is one emitted declaration plus its name family. Relatives (overloads, the
// members of one widget family) share a Group so they stay in one file.
type Chunk struct {
	Group string // name family the part file is named after
	Owner string // disambiguator basis when a group is split across parts
	Text  string
}

// Lines counts the chunk's lines, trailing blank line included (as committed).
func (c Chunk) Lines() int {
	n := 1
	for i := 0; i < len(c.Text); i++ {
		if c.Text[i] == '\n' {
			n++
		}
	}
	return n
}

// Part is the chunk set of one output file. Name is the sanitized file suffix;
// a caller writing a small output keeps its base file name instead of a suffix.
type Part struct {
	Chunks []Chunk
	Name   string
}

func (p *Part) Lines() int {
	n := 0
	for _, c := range p.Chunks {
		n += c.Lines()
	}
	return n
}

// Body is the file's declarations, in chunk order.
func (p *Part) Body() string {
	var b strings.Builder
	for _, c := range p.Chunks {
		b.WriteString(c.Text)
	}
	return b.String()
}

// Plan packs chunks into ordered parts. Sorting is a total order (group, owner, text),
// so the plan is identical for any input permutation and across consecutive generation
// runs. targetLines is the fill aim per part; maxLines is the ceiling a same-group piece
// must fit into, except for a single indivisible chunk over it.
func Plan(chunks []Chunk, targetLines, maxLines int) []Part {
	if len(chunks) == 0 {
		return nil
	}
	sorted := append([]Chunk(nil), chunks...)
	sort.Slice(sorted, func(i, j int) bool {
		a, b := sorted[i], sorted[j]
		if a.Group != b.Group {
			return a.Group < b.Group
		}
		if a.Owner != b.Owner {
			return a.Owner < b.Owner
		}
		return a.Text < b.Text
	})
	var parts []*Part
	nameKeys := map[*Part]string{}
	for i := 0; i < len(sorted); {
		group := sorted[i].Group
		j := i
		for j < len(sorted) && sorted[j].Group == group {
			j++
		}
		for _, piece := range pieces(sorted[i:j], maxLines) {
			var last *Part
			if len(parts) > 0 {
				last = parts[len(parts)-1]
			}
			// The part must own its chunk array: a piece is a sub-slice of sorted, and
			// appending later pieces to the part must not write into sorted's backing
			// array over chunks not yet processed.
			owned := append([]Chunk(nil), piece...)
			switch {
			case last != nil && last.Lines() > 0 && last.Lines()+lines(piece) > targetLines:
				parts = append(parts, &Part{Chunks: owned})
			case last != nil:
				last.Chunks = append(last.Chunks, owned...)
			default:
				parts = append(parts, &Part{Chunks: owned})
			}
			if _, ok := nameKeys[parts[len(parts)-1]]; !ok {
				nameKeys[parts[len(parts)-1]] = group
			}
		}
		i = j
	}
	nameParts(parts, nameKeys)
	out := make([]Part, len(parts))
	for i, p := range parts {
		out[i] = *p
	}
	return out
}

// pieces splits a same-group chunk list over the ceiling at owner boundaries, then at
// chunk boundaries, so every piece stays within the ceiling; a single chunk over it
// stays alone (indivisible).
func pieces(group []Chunk, maxLines int) [][]Chunk {
	if lines(group) <= maxLines {
		return [][]Chunk{group}
	}
	var out [][]Chunk
	var cur []Chunk
	for _, c := range group {
		if len(cur) > 0 && (cur[0].Owner != c.Owner || lines(cur)+c.Lines() > maxLines) {
			out = append(out, cur)
			cur = nil
		}
		cur = append(cur, c)
	}
	if len(cur) > 0 {
		out = append(out, cur)
	}
	return out
}

// nameParts names same-basis parts: the largest keeps the plain group name, the rest get
// their first chunk's owner as a disambiguator, then a counter as the last resort.
func nameParts(parts []*Part, nameKeys map[*Part]string) {
	var keys []string
	byName := map[string][]*Part{}
	for _, p := range parts {
		k := Sanitize(nameKeys[p])
		if _, ok := byName[k]; !ok {
			keys = append(keys, k)
		}
		byName[k] = append(byName[k], p)
	}
	used := map[string]bool{}
	for _, k := range keys {
		same := byName[k]
		sort.SliceStable(same, func(i, j int) bool { return same[i].Lines() > same[j].Lines() })
		for i, p := range same {
			s := k
			if i > 0 {
				s = k + "_" + Sanitize(p.Chunks[0].Owner)
			}
			for n := 2; used[s]; n++ {
				s = fmt.Sprintf("%s_%d", k, n)
			}
			used[s] = true
			p.Name = s
		}
	}
}

// Sanitize lowercases and maps every character outside [a-z0-9_] to '_'.
func Sanitize(name string) string {
	var b strings.Builder
	for i := 0; i < len(name); i++ {
		c := name[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		if c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '_' {
			b.WriteByte(c)
		} else {
			b.WriteByte('_')
		}
	}
	return b.String()
}

func lines(chunks []Chunk) int {
	n := 0
	for _, c := range chunks {
		n += c.Lines()
	}
	return n
}
