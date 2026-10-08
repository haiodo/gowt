package gowt

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/haiodo/gowt/icons/lucide"
)

func TestIconDictionaryMatchesLucideSet(t *testing.T) {
	want := map[string]bool{}
	for n, e := range iconDict {
		if _, ok := lucide.SVG(e.lucide); !ok {
			t.Errorf("%s: no embedded lucide icon %q", n, e.lucide)
		}
		if e.fd == "" || e.sf == "" || e.fluent == 0 {
			t.Errorf("%s: incomplete mapping %+v", n, e)
		}
		want[e.lucide] = true
	}
	for _, n := range lucide.Names() {
		if !want[n] {
			t.Errorf("embedded lucide icon %q is not in the dictionary", n)
		}
	}
}

func TestTintAlpha(t *testing.T) {
	img := tintAlpha(2, RGB{1, 2, 3}, func(x, y int) uint8 { return uint8(x*10 + y) })
	if p := img.NRGBAAt(1, 1); p.R != 1 || p.B != 3 || p.A != 11 {
		t.Fatalf("got %+v", p)
	}
}

func TestFindSymbolic(t *testing.T) {
	dir := t.TempDir()
	d := filepath.Join(dir, "Test", "symbolic", "actions")
	os.MkdirAll(d, 0o755)
	os.WriteFile(filepath.Join(dir, "Test", "index.theme"), []byte("[Icon Theme]\nInherits=Base,hicolor\n"), 0o644)
	os.WriteFile(filepath.Join(d, "x-symbolic.svg"), []byte(`<svg fill="#bebebe"/>`), 0o644)
	got := findSymbolic([]string{dir}, []string{"Test"}, "x")
	if !strings.HasSuffix(got, "x-symbolic.svg") {
		t.Fatalf("got %q", got)
	}
	if inh := themeParents(filepath.Join(dir, "Test")); len(inh) != 2 || inh[0] != "Base" {
		t.Fatalf("inherits %v", inh)
	}
}
