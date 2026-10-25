//go:build darwin

package cg

import (
	"fmt"
	"testing"

	"github.com/haiodo/gowt/gfx"
)

// BenchmarkParallelByOp finds which op kind stops scaling across goroutines: the mid view
// (about 270 nodes per tile) with one op kind left in, 1 and 8 workers.
func BenchmarkParallelByOp(b *testing.B) {
	d := doc(10000)
	v := views[1]
	zoom := v.zoom(d)
	o := v.origin(d, zoom)
	only := func(kinds ...gfx.OpKind) uint32 {
		m := ^uint32(0)
		for _, k := range kinds {
			m &^= 1 << k
		}
		return m
	}
	cases := []struct {
		name   string
		skip   uint32
		noSprt bool
	}{
		{"all", 0, false},
		{"all-liveblur", 0, true},
		{"shadow-liveblur", only(gfx.OpShadowBox, gfx.OpSave, gfx.OpRestore, gfx.OpTranslate), true},
		{"shadow", only(gfx.OpShadowBox, gfx.OpSave, gfx.OpRestore, gfx.OpTranslate), false},
		{"stroke", only(gfx.OpStroke, gfx.OpSave, gfx.OpRestore, gfx.OpTranslate), false},
		{"fill", only(gfx.OpFill, gfx.OpSave, gfx.OpRestore, gfx.OpTranslate), false},
		{"text", only(gfx.OpText, gfx.OpSave, gfx.OpRestore, gfx.OpTranslate), false},
		{"none", only(), false},
	}
	for _, c := range cases {
		for _, w := range []int{1, 8} {
			b.Run(fmt.Sprintf("%s/workers=%d", c.name, w), func(b *testing.B) {
				r := NewRenderer(Options{TextCache: true, ShadowSprite: !c.noSprt})
				r.skip = c.skip
				surfs := newSurfs(r, w)
				tileGrid(r, d, o, zoom*2, 2, w, surfs, nil)
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					tileGrid(r, d, o, zoom*2, 2, w, surfs, nil)
				}
			})
		}
	}
}
