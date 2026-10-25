//go:build darwin

package cg

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/haiodo/gowt/gfx"
	"github.com/haiodo/gowt/gfx/pen"
)

// refBox is one row of <id>.bounds.json: the bounds pen.dev resolved for a node, in its parent's space.
type refBox struct {
	ID, Type, Name string
	X, Y, W, H     float64
	Depth          int
	Off            bool // disabled itself or under a disabled node: not drawn, no layout to compare
}

// TestPenDiff compares the layout of gfx/pen with the bounds pen.dev resolved. PEN_FILE is the design,
// PEN_REF the directory with <screen id>.bounds.json files (PEN_SCREENS limits them to a comma
// separated list of ids). With PEN_OUT it also writes our render as <id>.ours.png and the sorted
// differences as <id>.diff.txt there. Skipped without PEN_FILE and PEN_REF.
func TestPenDiff(t *testing.T) {
	path, ref := os.Getenv("PEN_FILE"), os.Getenv("PEN_REF")
	if path == "" || ref == "" {
		t.Skip("PEN_FILE and PEN_REF not set")
	}
	out := os.Getenv("PEN_OUT")
	r := NewRenderer(Options{TextCache: true, ShadowSprite: true})
	d, err := pen.Open(path, pen.Options{Measurer: r})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("system font fallback: %v", d.FontFallback())
	want := map[string]bool{}
	for _, id := range strings.Split(os.Getenv("PEN_SCREENS"), ",") {
		if id != "" {
			want[id] = true
		}
	}
	var tot, totOK int
	causes := map[string]int{}
	files, _ := filepath.Glob(filepath.Join(ref, "*.bounds.json"))
	for _, f := range files {
		in := struct {
			ID, Name string
			W, H     float64
		}{ID: strings.TrimSuffix(filepath.Base(f), ".bounds.json")}
		if len(want) > 0 && !want[in.ID] {
			continue
		}
		raw, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		var rows []refBox
		if err := json.Unmarshal(raw, &rows); err != nil {
			t.Fatal(err)
		}
		s := gfx.NewScene()
		var sc *pen.Screen
		s.Update(func(tx *gfx.Tx) { sc, err = d.BuildNode(tx, nil, in.ID) })
		if err != nil {
			t.Fatal(err)
		}
		in.Name, in.W, in.H = sc.Name, sc.W, sc.H
		s.Commit()
		ours := map[string]pen.Box{}
		for _, b := range sc.Boxes() {
			ours[b.ID] = b
		}
		type diff struct {
			ref   refBox
			ours  pen.Box
			err   float64
			cause string
		}
		var diffs []diff
		var missing, ok, overflow, off int
		for i, rb := range rows {
			ob, found := ours[rb.ID]
			if !found && rb.Off {
				off++
				continue
			}
			if !found {
				missing++
				diffs = append(diffs, diff{ref: rb, err: 1e9, cause: "missing:" + rb.Type})
				continue
			}
			if ob.Type == "text" && ob.TextW > ob.W+0.5 {
				overflow++
			}
			if i == 0 {
				rb.X, rb.Y = 0, 0
			}
			dx, dy, dw, dh := math.Abs(ob.X-rb.X), math.Abs(ob.Y-rb.Y), math.Abs(ob.W-rb.W), math.Abs(ob.H-rb.H)
			e := max(dx, dy, dw, dh)
			if e <= 1 {
				ok++
				continue
			}
			c := rb.Type + ":"
			for _, p := range []struct {
				n string
				v float64
			}{{"x", dx}, {"y", dy}, {"w", dw}, {"h", dh}} {
				if p.v > 1 {
					c += p.n
				}
			}
			diffs = append(diffs, diff{rb, ob, e, c})
		}
		extra := 0
		have := map[string]bool{}
		for _, rb := range rows {
			have[rb.ID] = true
		}
		var extras []string
		for _, b := range sc.Boxes() {
			if !have[b.ID] {
				extra++
				extras = append(extras, fmt.Sprintf("%s %q %s %.2f %.2f %.2f %.2f", b.ID, b.Name, b.Type, b.X, b.Y, b.W, b.H))
			}
		}
		tot += len(rows) - off
		totOK += ok
		t.Logf("%s %q: %d nodes (%d disabled skipped), %d within 1px, %d off, %d missing here, %d extra here, %d text overflow",
			in.ID, in.Name, len(rows), off, ok, len(diffs)-missing, missing, extra, overflow)
		for _, df := range diffs {
			causes[df.cause]++
		}
		if out == "" {
			continue
		}
		sort.SliceStable(diffs, func(i, j int) bool { return diffs[i].err > diffs[j].err })
		var sb strings.Builder
		for _, df := range diffs {
			fmt.Fprintf(&sb, "%-9s err=%7.2f %s %q\n   ref  %.2f %.2f %.2f %.2f\n   ours %.2f %.2f %.2f %.2f\n",
				df.cause, df.err, df.ref.ID, df.ref.Name, df.ref.X, df.ref.Y, df.ref.W, df.ref.H, df.ours.X, df.ours.Y, df.ours.W, df.ours.H)
		}
		sb.WriteString("\nonly in ours:\n" + strings.Join(extras, "\n") + "\n")
		if err := os.WriteFile(filepath.Join(out, in.ID+".diff.txt"), []byte(sb.String()), 0o644); err != nil {
			t.Fatal(err)
		}
		img, err := gfx.RenderSubtree(r, s, sc.Root, gfx.R(0, 0, in.W, in.H), 2, gfx.Color{R: 1, G: 1, B: 1, A: 1})
		if err != nil {
			t.Fatal(err)
		}
		png, err := img.PNG()
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(out, in.ID+".ours.png"), png, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	keys := make([]string, 0, len(causes))
	for k := range causes {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return causes[keys[i]] > causes[keys[j]] })
	for _, k := range keys {
		t.Logf("  %-14s %d", k, causes[k])
	}
	t.Logf("total: %d of %d nodes within 1px", totOK, tot)
}
