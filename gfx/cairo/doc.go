// Package cairo replays gfx Pictures with cairo and Pango into image surfaces on Linux, without a
// window, a display or GTK: only libcairo, libpango, libpangocairo, libgobject and fontconfig are
// loaded, with purego. It also holds a CPU Host that composites tiles into a cairo context. It is
// empty on other systems.
//
// The C bindings are written from the public cairo, Pango and fontconfig API documentation.
//
// Experimental, like package gfx.
package cairo
