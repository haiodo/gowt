// Hand-written stand-ins for the helpers the translated graphics classes call but that are not
// ported: DPIUtil's scaling math, image loading (ImageLoader + codecs), strict checks.
package swt

import (
	"math"

	"github.com/haiodo/gowt/internal/jrt"
)

func DPIUtilGetDeviceZoom() int32 { return dpiNativeDeviceZoom }

func DPIUtilPixelToPoint(size int32, zoom int32) int32 {
	if zoom == 100 || size == SWTDEFAULT {
		return size
	}
	return int32(math.Round(float64(size) * 100 / float64(zoom)))
}

func DPIUtilPointToPixel(size int32, zoom int32) int32 {
	if zoom == 100 || size == SWTDEFAULT {
		return size
	}
	return int32(math.Round(float64(size) * float64(zoom) / 100))
}

// Only reached for zooms other than 100 on an image that has no representation at that zoom.
func DPIUtilScaleImageData(device *Device, imageData *ImageData, targetZoom int32, currentZoom int32) *ImageData {
	if targetZoom == currentZoom {
		return imageData
	}
	return imageData.ScaledTo(DPIUtilPointToPixel(DPIUtilPixelToPoint(imageData.Width, currentZoom), targetZoom),
		DPIUtilPointToPixel(DPIUtilPixelToPoint(imageData.Height, currentZoom), targetZoom))
}

func DPIUtilValidateLinearScaling(provider ImageDataProvider) {}

func DPIUtilValidateAndGetImagePathAtZoom(provider ImageFileNameProvider, zoom int32) any {
	panic("stub: image files are not supported (ImageLoader not ported)")
}

func CompatibilityCeil(p int32, q int32) int32 { return (p + q - 1) / q }

func StrictChecksRunIfStrictChecksEnabled(r jrt.Runnable)    {}
func StrictChecksRunWithStrictChecksDisabled(r jrt.Runnable) { r.Run() }

const FileFormatDEFAULT_ZOOM int32 = 100

// ImageDataLoader: every entry point needs ImageLoader's codecs.
func ImageDataLoaderLoad(source any) *ImageData { panic("stub: ImageLoader not ported") }
func ImageDataLoaderLoadByZoom(source any, fileZoom int32, targetZoom int32) any {
	panic("stub: ImageLoader not ported")
}
func ImageDataLoaderLoadBySize(source any, width int32, height int32) *ImageData {
	panic("stub: ImageLoader not ported")
}
func ImageDataLoaderCanLoadAtZoom(source any, fileZoom int32, targetZoom int32) bool { return false }
func ImageDataLoaderIsDynamicallySizable(source any) bool                            { return false }

// ImageColorTransformer.DEFAULT_DISABLED_IMAGE_TRANSFORMER: the default algorithm
// (forGrayscaledContrastBrightness(0.2, 2.9)).
type ImageColorTransformer struct{}

var ImageColorTransformerDEFAULT_DISABLED_IMAGE_TRANSFORMER = &ImageColorTransformer{}

func (t *ImageColorTransformer) AdaptPixelValue(red, green, blue, alpha int32) *RGBA {
	gray := min((77*red+151*green+28*blue)/255, 255)
	v := int32(min(max(0.2*(float32(gray)*2.9-128)+128, 0), 255))
	return NewRGBA(v, v, v, alpha)
}
