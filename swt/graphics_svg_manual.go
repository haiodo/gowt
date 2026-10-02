package swt

import (
	"bytes"
	"image"

	"github.com/haiodo/gowt/internal/jrt"
)

// SVGRasterizer is the Go counterpart of SWT's optional SVGRasterizer fragment: package
// github.com/haiodo/gowt/svg registers one on import. Without one an SVG is ERROR_UNSUPPORTED_FORMAT.
type SVGRasterizer interface {
	// Size is the intrinsic size of the document in pixels at 100%.
	Size(svg []byte) (width, height float64, err error)
	// Rasterize draws the whole document scaled to width x height; the result is non-premultiplied.
	Rasterize(svg []byte, width, height int32) (*image.NRGBA, error)
}

var svgRasterizer SVGRasterizer

func SetSVGRasterizer(r SVGRasterizer) { svgRasterizer = r }

// maxSVGSniff covers "<?xml" and "<svg" after leading whitespace/BOM, as FileFormat.isFileFormat does.
const maxSVGSniff = 64

func isSVGHead(head []byte) bool {
	h := bytes.TrimSpace(bytes.TrimPrefix(head, []byte("\xef\xbb\xbf")))
	return bytes.HasPrefix(h, []byte("<?xml")) || bytes.HasPrefix(h, []byte("<svg"))
}

func isSVGData(data []byte) bool { return isSVGHead(data[:min(len(data), maxSVGSniff)]) }

// isSVGSource peeks a stream or reads the head of a file; a stream that cannot rewind is not sizable.
func isSVGSource(source any) bool {
	switch v := source.(type) {
	case string:
		if v == jrt.NullString {
			return false
		}
		stream := openImageFile(v)
		defer stream.Close()
		return isSVGSource(stream)
	case jrt.InputStream:
		head, ok := jrt.PeekBytes(v, maxSVGSniff)
		return ok && isSVGHead(head)
	}
	return false
}

func rasterizeSVG(data []byte, width, height int32) *ImageData {
	if svgRasterizer == nil {
		ErrorCodeThrowableDetail(ERROR_UNSUPPORTED_FORMAT, nil, " [No SVG rasterizer found]")
	}
	if width <= 0 || height <= 0 {
		ErrorCodeThrowableDetail(ERROR_INVALID_ARGUMENT, nil, " [Cannot rasterize SVG for width or height <= 0]")
	}
	img, err := svgRasterizer.Rasterize(data, width, height)
	if err != nil {
		ErrorCodeThrowable(ERROR_INVALID_IMAGE, err)
	}
	return nrgbaToImageData(img)
}

func rasterizeSVGAtPercent(data []byte, percent int32) *ImageData {
	if svgRasterizer == nil {
		ErrorCodeThrowableDetail(ERROR_UNSUPPORTED_FORMAT, nil, " [No SVG rasterizer found]")
	}
	w, h, err := svgRasterizer.Size(data)
	if err != nil {
		ErrorCodeThrowable(ERROR_INVALID_IMAGE, err)
	}
	return rasterizeSVG(data, int32(max(1, float64(percent)*w/100+0.5)), int32(max(1, float64(percent)*h/100+0.5)))
}

// nrgbaToImageData keeps colour and alpha apart: directToImageData goes through premultiplied
// color.Color values, which darkens translucent edges.
func nrgbaToImageData(img *image.NRGBA) *ImageData {
	b := img.Bounds()
	w, h := int32(b.Dx()), int32(b.Dy())
	d := ImageDataInternal_new(w, h, 24, NewPaletteDataRedMaskGreenMaskBlueMask(0xFF0000, 0xFF00, 0xFF),
		4, nil, 0, nil, nil, -1, -1, IMAGE_SVG, 0, 0, 0, 0)
	row := make([]int32, w)
	alpha := make([]int8, w)
	for y := int32(0); y < h; y++ {
		p := img.Pix[img.PixOffset(b.Min.X, b.Min.Y+int(y)):]
		for x := int32(0); x < w; x++ {
			row[x] = int32(p[4*x])<<16 | int32(p[4*x+1])<<8 | int32(p[4*x+2])
			alpha[x] = int8(p[4*x+3])
		}
		d.SetPixelsXYPutWidthPixelsStartIndex(0, y, w, row, 0)
		d.SetAlphas(0, y, w, alpha, 0)
	}
	return d
}

// ImageDataLoaderLoadByZoom is ImageDataLoader.loadByZoom (stream or file name): the raster formats
// decode at their own resolution, so the element is at fileZoom; an SVG is drawn at targetZoom.
func ImageDataLoaderLoadByZoom(source any, fileZoom int32, targetZoom int32) *DPIUtilElementAtZoom {
	if isSVGSource(source) {
		if targetZoom <= 0 {
			ErrorCodeThrowableDetail(ERROR_INVALID_ARGUMENT, nil, " [Cannot rasterize SVG for zoom <= 0]")
		}
		return NewDPIUtilElementAtZoom(rasterizeSVGAtPercent(readAllImageBytes(source), 100*targetZoom/fileZoom), targetZoom)
	}
	return NewDPIUtilElementAtZoom(ImageDataLoaderLoad(source), fileZoom)
}

func ImageDataLoaderLoadBySize(source any, width int32, height int32) *ImageData {
	if isSVGSource(source) {
		return rasterizeSVG(readAllImageBytes(source), width, height)
	}
	return ImageDataLoaderLoad(source)
}

func ImageDataLoaderCanLoadAtZoom(source any, fileZoom int32, targetZoom int32) bool {
	return fileZoom == targetZoom || isSVGSource(source)
}

func ImageDataLoaderIsDynamicallySizable(source any) bool { return isSVGSource(source) }

// NewImageDataFromNRGBA converts a non-premultiplied raster to 24-bit ImageData with an alpha plane.
func NewImageDataFromNRGBA(img *image.NRGBA) *ImageData { return nrgbaToImageData(img) }
