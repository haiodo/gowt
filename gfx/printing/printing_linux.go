//go:build linux

package printing

import (
	"sync"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/gfx"
	"github.com/haiodo/gowt/gfx/cairo"
)

var (
	rendOnce sync.Once
	rend     *cairo.Renderer
	rendErr  error
)

func nativePage(p *gowt.PrintPage) (gfx.Page, error) {
	rendOnce.Do(func() { rend, rendErr = cairo.NewRenderer(cairo.Options{}) })
	if rendErr != nil {
		return nil, rendErr
	}
	n := p.Native()
	return rend.OnContext(n.Handle, n.UnitsPerPoint), nil
}
