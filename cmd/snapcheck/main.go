// Command snapcheck compares the PNGs written by `controlexample -snap` with the committed
// references (tests/snapshots). Non-GUI. With -update it replaces the references instead.
package main

import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"sort"

	"github.com/haiodo/gowt/internal/snapcmp"
)

func main() {
	ref := flag.String("ref", "tests/snapshots", "reference directory")
	got := flag.String("got", "", "directory written by controlexample -snap")
	diffDir := flag.String("diff", "", "directory for diff images of failures")
	update := flag.Bool("update", false, "replace the references with -got")
	threshold := flag.Int("threshold", 16, "per-pixel channel delta (0..255) that counts as different")
	fraction := flag.Float64("fraction", 0.002, "allowed share of differing pixels per image")
	flag.Parse()
	if *got == "" {
		fmt.Fprintln(os.Stderr, "snapcheck: -got is required")
		os.Exit(2)
	}
	if *update {
		must(replace(*ref, *got))
		return
	}
	refMeta, err := os.ReadFile(filepath.Join(*ref, "meta.txt"))
	if err != nil {
		fmt.Println("FAIL: no references in", *ref, "- run make snap-update")
		os.Exit(1)
	}
	gotMeta, err := os.ReadFile(filepath.Join(*got, "meta.txt"))
	must(err)
	if string(refMeta) != string(gotMeta) {
		fmt.Printf("SKIP: environment differs from the references.\nreference:\n%s\nthis run:\n%s", refMeta, gotMeta)
		return
	}
	refs := pngs(*ref)
	failed := 0
	for _, name := range refs {
		if msg := check(filepath.Join(*ref, name), filepath.Join(*got, name), filepath.Join(*diffDir, name), *threshold, *fraction); msg != "" {
			fmt.Printf("FAIL %s: %s\n", name, msg)
			failed++
		}
	}
	known := map[string]bool{}
	for _, n := range refs {
		known[n] = true
	}
	for _, name := range pngs(*got) {
		if !known[name] {
			fmt.Printf("FAIL %s: no reference (make snap-update)\n", name)
			failed++
		}
	}
	fmt.Printf("%d snapshots, %d failed (threshold %d, fraction %g)\n", len(refs), failed, *threshold, *fraction)
	if failed > 0 {
		os.Exit(1)
	}
}

func check(refPath, gotPath, diffPath string, threshold int, fraction float64) string {
	a, err := load(refPath)
	if err != nil {
		return err.Error()
	}
	b, err := load(gotPath)
	if err != nil {
		return "missing in this run: " + err.Error()
	}
	r := snapcmp.Compare(a, b, threshold)
	if r.SizeMismatch {
		return fmt.Sprintf("size %v vs reference %v", b.Bounds().Size(), a.Bounds().Size())
	}
	if r.Within(fraction) {
		return ""
	}
	if filepath.Dir(diffPath) != "." {
		must(os.MkdirAll(filepath.Dir(diffPath), 0o755))
		f, err := os.Create(diffPath)
		must(err)
		must(png.Encode(f, r.Diff))
		must(f.Close())
	}
	return fmt.Sprintf("%d of %d px differ (%.3f%%), diff %s", r.Differing, r.Total, 100*r.Fraction(), diffPath)
}

func load(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return png.Decode(f)
}

func pngs(dir string) []string {
	m, _ := filepath.Glob(filepath.Join(dir, "*.png"))
	for i := range m {
		m[i] = filepath.Base(m[i])
	}
	sort.Strings(m)
	return m
}

func replace(ref, got string) error {
	if err := os.MkdirAll(ref, 0o755); err != nil {
		return err
	}
	old, _ := filepath.Glob(filepath.Join(ref, "*.png"))
	for _, f := range old {
		os.Remove(f)
	}
	names := append(pngs(got), "meta.txt")
	for _, n := range names {
		data, err := os.ReadFile(filepath.Join(got, n))
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(ref, n), data, 0o644); err != nil {
			return err
		}
	}
	fmt.Println("updated", len(names)-1, "snapshots in", ref)
	return nil
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "snapcheck:", err)
		os.Exit(2)
	}
}
