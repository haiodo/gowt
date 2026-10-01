// DPIUtil as win32 needs it. The process is system DPI aware (manifest, and Display.create sets the thread context):
// the native zoom is the system DPI, the device zoom is SWT's default "integer" autoscale of it. No runtime rescaling.
package swt

const (
	AutoscalingModeDISABLED AutoscalingMode = iota
	AutoscalingModeDISABLED_INHERITED
	AutoscalingModeENABLED
)

func DPIUtilGetZoomForAutoscaleProperty(nativeDeviceZoom int32) int32 {
	return max((nativeDeviceZoom+25)/100*100, 100)
}
func DPIUtilMapDPIToZoom(dpi int32) int32          { return dpi * 100 / 96 }
func DPIUtilMapZoomToDPI(zoom int32) int32         { return zoom * 96 / 100 }
func DPIUtilGetScalingFactor(zoom int32) float32   { return float32(zoom) / 100 }
func DPIUtilSetMonitorSpecificScaling(active bool) {}
func DPIUtilIsMonitorSpecificScalingActive() bool  { return false }
func DPIUtilIsCustomAutoScale() bool               { return false }

func DPIUtilPixelToPointFloat(size float32, zoom int32) float32 {
	if zoom == 100 {
		return size
	}
	return size * 100 / float32(zoom)
}

func DPIUtilPixelToPointDouble(size float64, zoom int32) float64 {
	if zoom == 100 {
		return size
	}
	return size * 100 / float64(zoom)
}

func DPIUtilScaleImageDataElement(device *Device, element *DPIUtilElementAtZoom, targetZoom int32) *ImageData {
	return DPIUtilScaleImageData(device, element.Element().(*ImageData), targetZoom, element.Zoom())
}

func DPIUtilGetScalingFactorZooms(targetZoom int32, currentZoom int32) float32 {
	return float32(targetZoom) / float32(currentZoom)
}

// DPIUtilValidateAndGetImageDataAtZoom: the data at zoom, else at 150/200 (above 100 only), else 100.
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
	ErrorCodeThrowableDetail(ERROR_INVALID_ARGUMENT, nil, ": ImageDataProvider returns null ImageData at 100% zoom.")
	return nil
}
