// Hand-written stand-ins for the helpers the translated graphics classes call but that are not
// ported: DPIUtil's scaling math, image loading (ImageLoader + codecs), strict checks.
package swt

import (
	"fmt"
	"math"

	"github.com/haiodo/gowt/internal/jrt"
)

func DPIUtilGetDeviceZoom() int32 { return dpiNativeDeviceZoom }

func DPIUtilPixelToPoint(size int32, zoom int32) int32 {
	if zoom == 100 || size == DEFAULT {
		return size
	}
	return int32(math.Round(float64(size) * 100 / float64(zoom)))
}

func DPIUtilPointToPixel(size int32, zoom int32) int32 {
	if zoom == 100 || size == DEFAULT {
		return size
	}
	return int32(math.Round(float64(size) * float64(zoom) / 100))
}

// DPIUtil.scaleImageData(device, ImageData, targetZoom, currentZoom): only reached for zooms other than 100.
func DPIUtilScaleImageData(device *Device, imageData *ImageData, target int32, current int32) *ImageData {
	if imageData == nil || target == current {
		return imageData
	}
	return imageData.ScaledTo(DPIUtilPointToPixel(DPIUtilPixelToPoint(imageData.Width, current), target),
		DPIUtilPointToPixel(DPIUtilPixelToPoint(imageData.Height, current), target))
}

func DPIUtilScaleImageDataElement(device *Device, e *DPIUtilElementAtZoom, target int32) *ImageData {
	return DPIUtilScaleImageData(device, e.Element().(*ImageData), target, e.Zoom())
}

// DPIUtil.autoScaleImageData(device, data, dataZoom): scale to the device zoom.
func DPIUtilAutoScaleImageData(device *Device, imageData *ImageData, imageDataZoom int32) *ImageData {
	if imageData == nil || dpiNativeDeviceZoom == imageDataZoom {
		return imageData
	}
	return DPIUtilScaleImageData(device, imageData, dpiNativeDeviceZoom, imageDataZoom)
}

func DPIUtilMapDPIToZoom(dpi int32) int32 { return int32(math.Round(float64(dpi) * 100 / 96)) }

// DPIUtilValidateAndGetImageDataAtZoom: the data at zoom, else at 150/200 (above 100 only), else at 100.
func DPIUtilValidateAndGetImageDataAtZoom(provider ImageDataProvider, zoom int32) *DPIUtilElementAtZoom {
	if provider == nil {
		Error(ERROR_NULL_ARGUMENT)
	}
	candidates := []int32{zoom}
	if zoom > 100 && zoom <= 150 {
		candidates = append(candidates, 150)
	}
	if zoom > 100 {
		candidates = append(candidates, 200)
	}
	if zoom != 100 {
		candidates = append(candidates, 100)
	}
	for _, z := range candidates {
		if data := provider.GetImageData(z); data != nil {
			return NewDPIUtilElementAtZoom(data, z)
		}
	}
	ErrorCodeThrowableDetail(ERROR_INVALID_ARGUMENT, nil, fmt.Sprintf(": ImageDataProvider [%v] returns null ImageData at 100%% zoom.", provider))
	return nil
}

func DPIUtilValidateLinearScaling(provider ImageDataProvider) {}

// DPIUtilElementAtZoom is DPIUtil.ElementAtZoom<T>: the element is erased to any, callers cast it.
type DPIUtilElementAtZoom struct {
	element any
	zoom    int32
}

func NewDPIUtilElementAtZoom(element any, zoom int32) *DPIUtilElementAtZoom {
	if jrt.IsNil(element) {
		Error(ERROR_NULL_ARGUMENT)
	}
	if zoom <= 0 {
		Error(ERROR_INVALID_ARGUMENT)
	}
	return &DPIUtilElementAtZoom{element, zoom}
}

func (e *DPIUtilElementAtZoom) Element() any { return e.element }
func (e *DPIUtilElementAtZoom) Zoom() int32  { return e.zoom }

// DPIUtilValidateAndGetImagePathAtZoom: the path at zoom, else 150/200 (above 100 only), else 100.
func DPIUtilValidateAndGetImagePathAtZoom(provider ImageFileNameProvider, zoom int32) *DPIUtilElementAtZoom {
	if provider == nil {
		Error(ERROR_NULL_ARGUMENT)
	}
	candidates := []int32{zoom}
	if zoom > 100 && zoom <= 150 {
		candidates = append(candidates, 150)
	}
	if zoom > 100 {
		candidates = append(candidates, 200)
	}
	if zoom != 100 {
		candidates = append(candidates, 100)
	}
	for _, z := range candidates {
		if path := provider.GetImagePath(z); path != "" {
			return NewDPIUtilElementAtZoom(path, z)
		}
	}
	ErrorCodeThrowableDetail(ERROR_INVALID_ARGUMENT, nil, fmt.Sprintf(": ImageFileNameProvider [%v] returns null filename at 100%% zoom.", provider))
	return nil
}

func CompatibilityCeil(p int32, q int32) int32 { return (p + q - 1) / q }

func StrictChecksRunIfStrictChecksEnabled(r jrt.Runnable)    {}
func StrictChecksRunWithStrictChecksDisabled(r jrt.Runnable) { r.Run() }

const FileFormatDEFAULT_ZOOM int32 = 100

// ImageDataLoaderLoadByZoom is ImageDataLoader.loadByZoom (stream or file name): every supported format
// decodes at its own resolution, so the element is at fileZoom.
func ImageDataLoaderLoadByZoom(source any, fileZoom int32, targetZoom int32) *DPIUtilElementAtZoom {
	return NewDPIUtilElementAtZoom(ImageDataLoaderLoad(source), fileZoom)
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
