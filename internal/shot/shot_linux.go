// Package shot captures a Shell's window on linux for the snapshot commands (cmd/controlexample -snap, cmd/images).
package shot

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"slices"

	"github.com/haiodo/gowt/internal/gtk"
)

// Capture writes the toplevel window of widget as a PNG with its frame: a window-manager frame is copied
// from the root window, client-side decorations are in the window's own surface minus the shadow margin.
// The Frame is where the PNG lies on the screen, in logical points.
func Capture(widget int64, path string) (Frame, error) {
	win := gtk.GTK3Gtk_widget_get_window(gtk.GTK3Gtk_widget_get_toplevel(widget))
	if win == 0 {
		return Frame{}, fmt.Errorf("no GdkWindow")
	}
	// The window size is in logical points, the pixbuf copies device pixels: at scale 2 the logical size reads a quarter.
	scale := gtk.GTKGtk_widget_get_scale_factor(widget)
	w, h := gtk.GDKGdk_window_get_width(win), gtk.GDKGdk_window_get_height(win)
	var img *image.NRGBA
	var fe gtk.GdkRectangle
	gtk.GDKGdk_window_get_frame_extents(win, &fe)
	ox, oy := []int32{0}, []int32{0}
	if fe.Width > w || fe.Height > h {
		// A root window that reads back as one colour has no desktop content (seen under Xwayland): use the own surface.
		if root := readPixbuf(gtk.GDKGdk_get_default_root_window(), fe.X, fe.Y, fe.Width*scale, fe.Height*scale); root != nil && !uniform(root) {
			img = root
			ox[0], oy[0] = fe.X, fe.Y
		}
	}
	if img == nil {
		img = readPixbuf(win, 0, 0, w*scale, h*scale)
		gtk.GDKGdk_window_get_origin(win, ox, oy)
	}
	if img == nil {
		return Frame{}, fmt.Errorf("gdk_pixbuf_get_from_window failed")
	}
	// A Wayland window that was not painted yet reads back as all transparent; do not write that as a result.
	if !slices.ContainsFunc(img.Pix[3:], func(b byte) bool { return b != 0 }) {
		return Frame{}, fmt.Errorf("empty capture")
	}
	f, err := os.Create(path)
	if err != nil {
		return Frame{}, err
	}
	defer f.Close()
	out := trimShadow(img)
	if err := png.Encode(f, out); err != nil {
		return Frame{}, err
	}
	b, sc := out.Bounds(), float64(scale)
	return Frame{float64(ox[0]) + float64(b.Min.X)/sc, float64(oy[0]) + float64(b.Min.Y)/sc, float64(b.Dx()) / sc, float64(b.Dy()) / sc}, nil
}

// readPixbuf copies a w x h device pixel area of a GdkWindow at x, y; nil when GDK cannot read it.
func readPixbuf(win int64, x, y, w, h int32) *image.NRGBA {
	pb := gtk.GDKGdk_pixbuf_get_from_window(win, x, y, w, h)
	if pb == 0 {
		return nil
	}
	defer gtk.OSG_object_unref(pb)
	stride, n := gtk.GDKGdk_pixbuf_get_rowstride(pb), int32(4)
	if !gtk.GDKGdk_pixbuf_get_has_alpha(pb) {
		n = 3
	}
	buf := make([]int8, int(stride)*int(h))
	gtk.CMemmoveOverload8(buf, gtk.GDKGdk_pixbuf_get_pixels(pb), int64(len(buf)))
	img := image.NewNRGBA(image.Rect(0, 0, int(w), int(h)))
	for py := 0; py < int(h); py++ {
		for px := 0; px < int(w); px++ {
			si, dst := py*int(stride)+px*int(n), img.PixOffset(px, py)
			img.Pix[dst], img.Pix[dst+1], img.Pix[dst+2], img.Pix[dst+3] = byte(buf[si]), byte(buf[si+1]), byte(buf[si+2]), 255
			if n == 4 {
				img.Pix[dst+3] = byte(buf[si+3])
			}
		}
	}
	return img
}

func uniform(img *image.NRGBA) bool {
	first := img.Pix[:4]
	for i := 4; i+4 <= len(img.Pix); i += 4 {
		if !slices.Equal(img.Pix[i:i+4], first) {
			return false
		}
	}
	return true
}

// trimShadow keeps the box of the nearly opaque pixels, so the rounded corners stay.
func trimShadow(img *image.NRGBA) image.Image {
	b, box := img.Bounds(), image.Rectangle{}
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if img.Pix[img.PixOffset(x, y)+3] >= 0xF0 {
				box = box.Union(image.Rect(x, y, x+1, y+1))
			}
		}
	}
	if box.Empty() {
		return img
	}
	return img.SubImage(box)
}

// Meta is what the pixels depend on besides the code: snapcheck compares it before comparing images.
func Meta(widget int64) string {
	return fmt.Sprintf("os=linux\ngtk=%d.%d.%d\nscale=%d\nscreen=%dx%d\n", gtk.GTKGtk_get_major_version(), gtk.GTKGtk_get_minor_version(),
		gtk.GTKGtk_get_micro_version(), gtk.GTKGtk_widget_get_scale_factor(widget), gtk.GDKGdk_screen_width(), gtk.GDKGdk_screen_height())
}
