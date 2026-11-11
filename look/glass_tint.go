package look

import "github.com/haiodo/gowt"

const glassTintKey = "gowt.look.glassTint"

// SetGlassTint tints the glass of c so the content under it shows through less: macOS 26 sets the
// NSGlassEffectView tintColor, older macOS a denser material; the tint survives a glass reinstall.
func SetGlassTint(c Container, tint gowt.RGBA) { setGlassTint(c.AsComposite(), tint) }
