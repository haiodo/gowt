package main

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/haiodo/gowt/tooling/genplan"
)

// wkChunks builds n one-line chunks per family. 300 per family tops the 800-line
// target, so three families plan into two parts: [alpha,beta] and [gamma].
func wkChunks(perFamily int) []genplan.Chunk {
	var cs []genplan.Chunk
	for _, g := range []string{"alpha", "beta", "gamma"} {
		for i := 0; i < perFamily; i++ {
			cs = append(cs, genplan.Chunk{Group: g, Owner: g + "_owner", Text: fmt.Sprintf("\tf%s%d func(uintptr) uintptr\n", g, i)})
		}
	}
	return cs
}

func wkBinds(n int) []string {
	var b []string
	for i := 0; i < n; i++ {
		b = append(b, "\tbindOne(&f, \"f\")\n")
	}
	return b
}

func read(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}

func TestWriteOutDropsStalePartsWhenOutputShrinks(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "gen_funcs.go")

	writeOut(out, wkChunks(300), wkBinds(900))
	if _, err := os.Stat(filepath.Join(dir, "gen_funcs_alpha.go")); err != nil {
		t.Fatalf("multi-part run: no alpha part: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "gen_funcs_gamma.go")); err != nil {
		t.Fatalf("multi-part run: no gamma part: %v", err)
	}
	base := read(t, out)
	if !strings.Contains(base, "func bindAll()") || strings.Contains(base, "var (") {
		t.Fatalf("multi-part base should hold only bindAll:\n%s", base)
	}

	// The same output small enough for one part: stale parts must go, the base
	// keeps the combined layout.
	writeOut(out, wkChunks(3), wkBinds(9))
	matches, err := filepath.Glob(filepath.Join(dir, "gen_funcs_*.go"))
	if err != nil || len(matches) != 0 {
		t.Fatalf("stale parts left: %v (%v)", matches, err)
	}
	got := read(t, out)
	if !strings.Contains(got, "var (") || !strings.Contains(got, "func bindAll()") {
		t.Fatalf("single-part base lost its layout:\n%s", got)
	}
}

func TestWriteOutDeterministicAcrossChunkOrder(t *testing.T) {
	dir1, dir2 := t.TempDir(), t.TempDir()
	writeOut(filepath.Join(dir1, "gen_funcs.go"), wkChunks(300), wkBinds(900))
	cs := wkChunks(300)
	rand.New(rand.NewSource(7)).Shuffle(len(cs), func(i, j int) { cs[i], cs[j] = cs[j], cs[i] })
	writeOut(filepath.Join(dir2, "gen_funcs.go"), cs, wkBinds(900))
	d1, _ := os.ReadDir(dir1)
	d2, _ := os.ReadDir(dir2)
	if len(d1) != len(d2) {
		t.Fatalf("file sets differ: %d vs %d", len(d1), len(d2))
	}
	for i := range d1 {
		if d1[i].Name() != d2[i].Name() {
			t.Fatalf("file names differ: %s vs %s", d1[i].Name(), d2[i].Name())
		}
		if read(t, filepath.Join(dir1, d1[i].Name())) != read(t, filepath.Join(dir2, d2[i].Name())) {
			t.Fatalf("file %s differs between runs", d1[i].Name())
		}
	}
}
