// The PNG/GIF/BMP/JPEG codec backend is not translated from SWT's org.eclipse.swt.internal.image
// (FileFormat + PNGFileFormat/GIFFileFormat/WinBMPFileFormat/OS2BMPFileFormat/LZW*/Png*/LEData*):
// it is a hand-written wrapper over Go's stdlib image codecs plus golang.org/x/image/bmp instead.
// See tooling/j2go/README.md "Round 9 images" for why, and the differences this leaves from real
// SWT's own decoders. ImageLoader/ImageData/ImageLoaderEvent/ImageLoaderListener translate for
// real; this file only replaces their FileFormat/NativeImageLoader/ImageDataLoader entry points.
package swt

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"

	"github.com/haiodo/gowt/internal/jrt"
	"golang.org/x/image/bmp"
)

// ImageDataLoaderLoad replaces the graphics_stubs_manual.go stub: source is either a
// jrt.InputStream (ImageData(InputStream)) or a filename (ImageData(String)) - Java's two
// ImageDataLoader.load overloads collapse to this one Go func (Manual.staticMember has no
// per-overload dispatch for a manual type's static members).
func ImageDataLoaderLoad(source any) *ImageData {
	images := decodeImages(readAllImageBytes(source))
	if len(images) == 0 {
		Error(ERROR_UNSUPPORTED_FORMAT)
		return nil
	}
	return images[0]
}

// LoadByZoomStub replaces ImageLoader.loadByZoom(InputStream,int,int)'s body (Stream/Optional/
// DPIUtil.ElementAtZoom<T>, no translator rule; the HiDPI @2x-variant path is out of scope, see
// README "Round 9 images"): decodes every frame/page the source holds into this.Data.
func (this *ImageLoader) LoadByZoomStub(stream jrt.InputStream, fileZoom int32, targetZoom int32) *jrt.List {
	if stream == nil {
		Error(ERROR_NULL_ARGUMENT)
	}
	this.Reset()
	data := readAllImageBytes(stream)
	if g, err := gif.DecodeAll(bytes.NewReader(data)); err == nil {
		this.Data = gifToImageDatas(g)
		this.LogicalScreenWidth = int32(g.Config.Width)
		this.LogicalScreenHeight = int32(g.Config.Height)
		this.RepeatCount = int32(g.LoopCount)
	} else {
		this.Data = decodeImages(data)
	}
	if len(this.Data) == 0 {
		Error(ERROR_UNSUPPORTED_FORMAT)
	}
	list := jrt.NewList()
	for _, d := range this.Data {
		list.Add(d)
	}
	return list
}

// NativeImageLoaderSave replaces org.eclipse.swt.internal.NativeImageLoader.save (a cocoa PI
// file never translated) - encodes loader.Data[0] (loader.Data[1:] only for an animated GIF).
func NativeImageLoaderSave(stream jrt.OutputStream, format int32, loader *ImageLoader) {
	if len(loader.Data) == 0 {
		Error(ERROR_INVALID_ARGUMENT)
		return
	}
	w := jrt.AsWriter(stream)
	var err error
	switch format {
	case IMAGE_PNG:
		err = png.Encode(w, imageDataToImage(loader.Data[0]))
	case IMAGE_JPEG:
		err = jpeg.Encode(w, imageDataToImage(loader.Data[0]), nil)
	case IMAGE_BMP, IMAGE_BMP_RLE:
		err = bmp.Encode(w, imageDataToImage(loader.Data[0]))
	case IMAGE_GIF:
		err = gifEncode(w, loader)
	default:
		Error(ERROR_UNSUPPORTED_FORMAT)
		return
	}
	if err != nil {
		Error(ERROR_IO)
	}
}

// FileFormatIsDynamicallySizableFormat replaces FileFormat.isDynamicallySizableFormat: true only
// for a format loadable at an arbitrary requested size, which none of image/png|gif|jpeg|bmp are
// (all decode at their own stored resolution) - always false.
func FileFormatIsDynamicallySizableFormat(stream jrt.InputStream) bool { return false }

// FileFormatCanLoadAtZoom replaces FileFormat.canLoadAtZoom: only ever reached through an
// ElementAtZoom<InputStream> construction this port doesn't model (HiDPI @2x variants, out of
// scope - the call site already panics building that argument), so this body is unreachable.
func FileFormatCanLoadAtZoom(elementAtZoom any, targetZoom int32) bool { return false }

func readAllImageBytes(source any) []byte {
	var stream jrt.InputStream
	switch v := source.(type) {
	case string:
		stream = jrt.NewFileInputStream(v)
		defer stream.Close()
	case jrt.InputStream:
		stream = v
	default:
		panic("stub: ImageDataLoader.load: unsupported source")
	}
	data := jrt.ReadAllBytes(stream)
	out := make([]byte, len(data))
	for i, b := range data {
		out[i] = byte(b)
	}
	return out
}

// decodeImages tries GIF first (its own multi-frame API), then falls back to any single-frame
// format image.Decode's registered codecs (png/jpeg/bmp) recognize.
func decodeImages(data []byte) []*ImageData {
	if g, err := gif.DecodeAll(bytes.NewReader(data)); err == nil {
		return gifToImageDatas(g)
	}
	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil
	}
	return []*ImageData{imageToImageData(img, swtImageType(format), 0, 0, 0, 0)}
}

func swtImageType(format string) int32 {
	switch format {
	case "png":
		return IMAGE_PNG
	case "jpeg":
		return IMAGE_JPEG
	case "bmp":
		return IMAGE_BMP
	case "gif":
		return IMAGE_GIF
	default:
		return IMAGE_UNDEFINED
	}
}

func gifToImageDatas(g *gif.GIF) []*ImageData {
	images := make([]*ImageData, len(g.Image))
	for i, frame := range g.Image {
		disposal, delay := int32(0), int32(0)
		if i < len(g.Disposal) {
			disposal = int32(g.Disposal[i])
		}
		if i < len(g.Delay) {
			delay = int32(g.Delay[i])
		}
		b := frame.Bounds()
		images[i] = palettedToImageData(frame, IMAGE_GIF, int32(b.Min.X), int32(b.Min.Y), disposal, delay)
	}
	return images
}

func imageToImageData(img image.Image, type_, x, y, disposalMethod, delayTime int32) *ImageData {
	if p, ok := img.(*image.Paletted); ok {
		return palettedToImageData(p, type_, x, y, disposalMethod, delayTime)
	}
	return directToImageData(img, type_, x, y, disposalMethod, delayTime)
}

// palettedToImageData always uses 8-bit depth regardless of the real palette size (real SWT
// picks the smallest of 1/2/4/8 that fits) - simpler, and ImageData packs/unpacks either way.
func palettedToImageData(p *image.Paletted, type_, x, y, disposalMethod, delayTime int32) *ImageData {
	bounds := p.Bounds()
	w, h := int32(bounds.Dx()), int32(bounds.Dy())
	colors := make([]*RGB, len(p.Palette))
	transparentPixel := int32(-1)
	for i, c := range p.Palette {
		r, g, b, a := c.RGBA()
		colors[i] = NewRGB(int32(r>>8), int32(g>>8), int32(b>>8))
		if a == 0 && transparentPixel == -1 {
			transparentPixel = int32(i)
		}
	}
	img := ImageDataInternal_new(w, h, 8, NewPaletteData(colors), 4, nil, 0, nil, nil, -1,
		transparentPixel, type_, x, y, disposalMethod, delayTime)
	row := make([]int32, w)
	for yy := 0; yy < int(h); yy++ {
		off := p.PixOffset(bounds.Min.X, bounds.Min.Y+yy)
		for i, v := range p.Pix[off : off+int(w)] {
			row[i] = int32(v)
		}
		img.SetPixelsXYPutWidthPixelsStartIndex(0, int32(yy), w, row, 0)
	}
	return img
}

// directToImageData handles every non-paletted image.Image (RGBA/NRGBA/Gray/YCbCr/...) as 24-bit
// direct-color RGB (real SWT keeps 8-bit grayscale as an indexed gray-ramp palette instead - one
// converter here for every non-indexed source). It buffers each row before deciding whether to
// keep the alpha plane at all: a decoder's pixel type (NRGBA vs RGBA) says it CAN carry alpha,
// not that this image actually uses it (a 32bpp BMP decodes as NRGBA with every byte still 255).
func directToImageData(img image.Image, type_, x, y, disposalMethod, delayTime int32) *ImageData {
	bounds := img.Bounds()
	w, h := int32(bounds.Dx()), int32(bounds.Dy())
	result := ImageDataInternal_new(w, h, 24, NewPaletteDataRedMaskGreenMaskBlueMask(0xFF0000, 0xFF00, 0xFF),
		4, nil, 0, nil, nil, -1, -1, type_, x, y, disposalMethod, delayTime)
	pixels := make([][]int32, h)
	alphas := make([][]int8, h)
	hasAlpha := false
	for yy := int32(0); yy < h; yy++ {
		pixels[yy] = make([]int32, w)
		alphas[yy] = make([]int8, w)
		for xx := int32(0); xx < w; xx++ {
			r, g, b, a := img.At(bounds.Min.X+int(xx), bounds.Min.Y+int(yy)).RGBA()
			pixels[yy][xx] = (int32(r>>8) << 16) | (int32(g>>8) << 8) | int32(b>>8)
			alphas[yy][xx] = int8(a >> 8)
			hasAlpha = hasAlpha || (a>>8) != 255
		}
		result.SetPixelsXYPutWidthPixelsStartIndex(0, yy, w, pixels[yy], 0)
	}
	if hasAlpha {
		for yy := int32(0); yy < h; yy++ {
			result.SetAlphas(0, yy, w, alphas[yy], 0)
		}
	}
	return result
}

// imageDataToImage is directToImageData/palettedToImageData's inverse, for save(): an indexed
// ImageData becomes *image.Paletted, everything else (including 1/2/4-bit, which GetPixel/
// GetAlpha already unpack for us) becomes *image.NRGBA.
func imageDataToImage(d *ImageData) image.Image {
	w, h := int(d.Width), int(d.Height)
	if !d.Palette.IsDirect {
		pal := make(color.Palette, len(d.Palette.Colors))
		for i, c := range d.Palette.Colors {
			pal[i] = color.RGBA{R: uint8(c.Red), G: uint8(c.Green), B: uint8(c.Blue), A: 255}
		}
		img := image.NewPaletted(image.Rect(0, 0, w, h), pal)
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				img.SetColorIndex(x, y, uint8(d.GetPixel(int32(x), int32(y))))
			}
		}
		return img
	}
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			pixel := d.GetPixel(int32(x), int32(y))
			a := d.GetAlpha(int32(x), int32(y))
			img.SetNRGBA(x, y, color.NRGBA{
				R: uint8((pixel >> 16) & 0xFF), G: uint8((pixel >> 8) & 0xFF), B: uint8(pixel & 0xFF), A: uint8(a),
			})
		}
	}
	return img
}

// gifEncode writes loader.Data as an (animated, if >1 frame) GIF. A frame that came from a
// direct-color source (imageDataToImage returns *image.NRGBA) needs quantizing first - gif.Encode
// does that itself for a single frame, but gif.EncodeAll's API requires already-paletted frames.
func gifEncode(w interface{ Write([]byte) (int, error) }, loader *ImageLoader) error {
	if len(loader.Data) == 1 {
		return gif.Encode(w, imageDataToImage(loader.Data[0]), nil)
	}
	g := &gif.GIF{LoopCount: int(loader.RepeatCount)}
	for _, d := range loader.Data {
		img := imageDataToImage(d)
		pal, ok := img.(*image.Paletted)
		if !ok {
			pal = quantizeToPaletted(img)
		}
		g.Image = append(g.Image, pal)
		g.Delay = append(g.Delay, int(d.DelayTime))
		g.Disposal = append(g.Disposal, byte(d.DisposalMethod))
	}
	return gif.EncodeAll(w, g)
}

// quantizeToPaletted's nearest-256-color reduction is coarse (no dithering, no popularity
// analysis) - only exercised by a direct-color ImageData saved as a multi-frame GIF, which SWT
// itself does not support either (GIF requires an indexed palette per frame).
func quantizeToPaletted(img image.Image) *image.Paletted {
	bounds := img.Bounds()
	pal := color.Palette{}
	seen := map[color.Color]uint8{}
	out := image.NewPaletted(bounds, nil)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			c := color.RGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: uint8(a >> 8)}
			idx, ok := seen[c]
			if !ok && len(pal) < 256 {
				idx = uint8(len(pal))
				pal = append(pal, c)
				seen[c] = idx
			} else if !ok {
				idx = uint8(pal.Index(c))
			}
			out.Pix[out.PixOffset(x, y)] = idx
		}
	}
	out.Palette = pal
	return out
}
