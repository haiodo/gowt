// Package shot captures a Shell's window on linux for the snapshot commands (cmd/controlexample -snap, cmd/images).
package shot

import (
	"fmt"
	"image"
	"image/png"
	"os"

	"github.com/haiodo/gowt/internal/gtk"
)

// WindowPNG writes the toplevel window that holds widget as a PNG: the client area as the X server shows
// it (no window-manager frame), read back through GDK.
func WindowPNG(widget int64, path string) error {
	win := gtk.GTK3Gtk_widget_get_window(gtk.GTK3Gtk_widget_get_toplevel(widget))
	if win == 0 {
		return fmt.Errorf("no GdkWindow")
	}
	w, h := gtk.GDKGdk_window_get_width(win), gtk.GDKGdk_window_get_height(win)
	pb := gtk.GDKGdk_pixbuf_get_from_window(win, 0, 0, w, h)
	if pb == 0 {
		return fmt.Errorf("gdk_pixbuf_get_from_window failed")
	}
	defer gtk.OSG_object_unref(pb)
	stride, n := gtk.GDKGdk_pixbuf_get_rowstride(pb), int32(4)
	if !gtk.GDKGdk_pixbuf_get_has_alpha(pb) {
		n = 3
	}
	buf := make([]int8, int(stride)*int(h))
	gtk.CMemmoveOverload8(buf, gtk.GDKGdk_pixbuf_get_pixels(pb), int64(len(buf)))
	img := image.NewNRGBA(image.Rect(0, 0, int(w), int(h)))
	for y := 0; y < int(h); y++ {
		for x := 0; x < int(w); x++ {
			src, dst := y*int(stride)+x*int(n), img.PixOffset(x, y)
			img.Pix[dst], img.Pix[dst+1], img.Pix[dst+2], img.Pix[dst+3] = byte(buf[src]), byte(buf[src+1]), byte(buf[src+2]), 255
			if n == 4 {
				img.Pix[dst+3] = byte(buf[src+3])
			}
		}
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

// Meta is what the pixels depend on besides the code: snapcheck compares it before comparing images.
func Meta(widget int64) string {
	return fmt.Sprintf("os=linux\ngtk=%d.%d.%d\nscale=%d\nscreen=%dx%d\n", gtk.GTKGtk_get_major_version(), gtk.GTKGtk_get_minor_version(),
		gtk.GTKGtk_get_micro_version(), gtk.GTKGtk_widget_get_scale_factor(widget), gtk.GDKGdk_screen_width(), gtk.GDKGdk_screen_height())
}
