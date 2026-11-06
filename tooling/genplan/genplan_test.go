package genplan

import (
	"math/rand"
	"strings"
	"testing"
)

func chunk(group, owner, text string) Chunk {
	return Chunk{Group: group, Owner: owner, Text: text}
}

func TestPlanDeterministicAcrossInputPermutations(t *testing.T) {
	mk := func() []Chunk {
		var cs []Chunk
		for _, g := range []string{"tree", "widget", "window", "glib"} {
			for i := 0; i < 20; i++ {
				cs = append(cs, chunk(g, g+"_owner", "func "+g+string(rune('a'+i%26))+"(x int) int { return x }\n\n"))
			}
		}
		return cs
	}
	base := Plan(mk(), 100, 200)
	for seed := int64(0); seed < 20; seed++ {
		cs := mk()
		rand.New(rand.NewSource(seed)).Shuffle(len(cs), func(i, j int) { cs[i], cs[j] = cs[j], cs[i] })
		got := Plan(cs, 100, 200)
		if len(got) != len(base) {
			t.Fatalf("seed %d: %d parts, want %d", seed, len(got), len(base))
		}
		for i := range base {
			if got[i].Name != base[i].Name || got[i].Body() != base[i].Body() {
				t.Fatalf("seed %d: part %d differs: %q vs %q", seed, i, got[i].Name, base[i].Name)
			}
		}
	}
}

func TestPlanGroupsStayTogetherAndPartsUnderTarget(t *testing.T) {
	var cs []Chunk
	for _, g := range []string{"aaa", "bbb", "ccc"} {
		for i := 0; i < 10; i++ {
			cs = append(cs, chunk(g, g, "func f() {}\n\n"))
		}
	}
	parts := Plan(cs, 25, 50)
	for _, p := range parts {
		if p.Lines() > 50 {
			t.Errorf("part %q: %d lines over ceiling 50", p.Name, p.Lines())
		}
		group := p.Chunks[0].Group
		for _, c := range p.Chunks {
			if c.Group != group {
				t.Errorf("part %q mixes groups %q and %q", p.Name, group, c.Group)
			}
		}
	}
}

func TestPlanIndivisibleChunkOverCeilingStaysAlone(t *testing.T) {
	big := "func huge() {\n" + strings.Repeat("\tx()\n", 60) + "}\n\n" // 62 lines over a 50 ceiling
	cs := []Chunk{chunk("small", "o", "func s() {}\n\n"), chunk("huge", "o", big)}
	parts := Plan(cs, 50, 50)
	if len(parts) != 2 {
		t.Fatalf("got %d parts, want 2", len(parts))
	}
	for _, p := range parts {
		if p.Name != "small" && p.Name != "huge" {
			t.Errorf("part name %q not a group name", p.Name)
		}
	}
}

func TestPlanSplitGroupDisambiguatesByOwner(t *testing.T) {
	var cs []Chunk
	for _, o := range []string{"tree_view", "tree_store"} {
		for i := 0; i < 20; i++ {
			cs = append(cs, chunk("tree", o, "func t"+o+string(rune('a'+i))+"() {}\n\n"))
		}
	}
	parts := Plan(cs, 30, 30) // group over the ceiling -> per-owner pieces kept apart by the target
	if len(parts) < 2 {
		t.Fatalf("got %d parts, want the split group spread over several", len(parts))
	}
	names := map[string]bool{}
	for _, p := range parts {
		names[p.Name] = true
		if !strings.HasPrefix(p.Name, "tree") {
			t.Errorf("part name %q not based on its group", p.Name)
		}
	}
	if len(names) != len(parts) {
		t.Errorf("part names not unique: %v", names)
	}
}

func TestPlanLargestKeepsPlainName(t *testing.T) {
	// Same-name pieces arise when one group splits over the ceiling and the target then
	// keeps the pieces apart: the largest keeps the plain group name, the rest get their
	// first chunk's owner as a disambiguator.
	var cs []Chunk
	for i := 0; i < 8; i++ {
		cs = append(cs, chunk("dup", "big", "func b() {}\n\n\n\n\n"))
	}
	for i := 0; i < 4; i++ {
		cs = append(cs, chunk("dup", "small", "func s() {}\n\n"))
	}
	parts := Plan(cs, 20, 20)
	names := map[string]bool{}
	for _, p := range parts {
		names[p.Name] = true
	}
	if !names["dup"] || !names["dup_big"] || !names["dup_small"] {
		t.Errorf("names %v, want plain name for the largest and owner for the rest", names)
	}
}

func TestPlanEveryChunkExactlyOnce(t *testing.T) {
	// The shape that bit once: a tiny first group whose part takes a sub-slice of the
	// sorted chunk array, followed by a group over the ceiling whose pieces merge into
	// that part - the merge must not duplicate or lose chunks across parts.
	var cs []Chunk
	cs = append(cs, chunk("cmemset", "cmemset", "func cmemset() {}\n\n"))
	for i := 0; i < 40; i++ {
		cs = append(cs, chunk("color", "color_chooser", "func color"+string(rune('a'+i%26))+string(rune('a'+i/26))+"() int { return "+string(rune('0'+i%10))+" }\n\n\n\n\n\n\n\n"))
	}
	for _, target := range []int{20, 60, 500} {
		parts := Plan(cs, target, 30)
		seen := map[string]int{}
		total := 0
		for _, p := range parts {
			for _, c := range p.Chunks {
				seen[c.Text]++
				total++
			}
		}
		if total != len(cs) {
			t.Fatalf("target %d: %d chunks in parts, want %d", target, total, len(cs))
		}
		for text, n := range seen {
			if n != 1 {
				t.Fatalf("target %d: chunk %q appears %d times", target, text, n)
			}
		}
	}
}

func TestSanitize(t *testing.T) {
	cases := map[string]string{
		"Tree_View": "tree_view",
		"GTK-IS":    "gtk_is",
		"a.b c":     "a_b_c",
	}
	for in, want := range cases {
		if got := Sanitize(in); got != want {
			t.Errorf("Sanitize(%q) = %q, want %q", in, got, want)
		}
	}
}
