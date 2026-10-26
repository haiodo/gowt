//go:build darwin

package printing

import (
	"sync"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/gfx"
	"github.com/haiodo/gowt/gfx/cg"
)

var (
	rendOnce sync.Once
	rend     *cg.Renderer
)

func nativePage(p *gowt.PrintPage) (gfx.Page, error) {
	rendOnce.Do(func() { rend = cg.NewRenderer(cg.Options{}) })
	n := p.Native()
	return rend.OnContext(n.Handle, n.UnitsPerPoint, n.Height), nil
}
