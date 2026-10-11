// DPIUtil as win32 needs it. Monitor-specific scaling follows upstream: on unless swt.autoScale.updateOnRuntime is
// "false", then Display.create puts the thread in PerMonitorV2 and Shell handles WM_DPICHANGED. Ceiling: images are
// always scaled with ImageData.ScaledTo, upstream's smooth method for this mode is not ported.
package swt

import (
	"math"
	"strconv"
	"strings"

	"github.com/haiodo/gowt/internal/jrt"
)

const (
	dpiAutoScaleProperty        = "swt.autoScale"
	dpiAutoScaleUpdateOnRuntime = "swt.autoScale.updateOnRuntime"
)

const (
	AutoscalingModeDISABLED AutoscalingMode = iota
	AutoscalingModeDISABLED_INHERITED
	AutoscalingModeENABLED
)

// DPIUtilGetZoomForAutoscaleProperty is AutoScaleCalculation.parseFrom(swt.autoScale).getAutoScaledZoom.
func DPIUtilGetZoomForAutoscaleProperty(nativeDeviceZoom int32) int32 {
	value := strings.ToLower(jrt.GetProperty(dpiAutoScaleProperty, ""))
	if value == "" || (value != "false" && value != "half" && value != "quarter" && value != "exact" && value != "integer") {
		if n, err := strconv.Atoi(value); err == nil {
			return int32(n)
		}
		if DPIUtilIsMonitorSpecificScalingActive() {
			value = "quarter"
		} else {
			value = "integer"
		}
	}
	switch value {
	case "false":
		return 100
	case "half":
		return int32(math.RoundToEven(float64(nativeDeviceZoom)/50)) * 50
	case "quarter":
		return int32(math.Round(float64(nativeDeviceZoom)/25)) * 25
	case "exact":
		return nativeDeviceZoom
	}
	return max((nativeDeviceZoom+25)/100*100, 100)
}
func DPIUtilMapDPIToZoom(dpi int32) int32        { return dpi * 100 / 96 }
func DPIUtilMapZoomToDPI(zoom int32) int32       { return zoom * 96 / 100 }
func DPIUtilGetScalingFactor(zoom int32) float32 { return float32(zoom) / 100 }
func DPIUtilIsCustomAutoScale() bool             { return jrt.GetProperty(dpiAutoScaleProperty, "") != "" }

func DPIUtilIsMonitorSpecificScalingActive() bool {
	return !strings.EqualFold(jrt.GetProperty(dpiAutoScaleUpdateOnRuntime, ""), "false")
}

func DPIUtilSetMonitorSpecificScaling(active bool) {
	if !active || jrt.GetProperty(dpiAutoScaleUpdateOnRuntime, "") != "force" {
		jrt.SetProperty(dpiAutoScaleUpdateOnRuntime, strconv.FormatBool(active))
	}
	if !active || jrt.GetProperty(dpiAutoScaleUpdateOnRuntime, "") == "force" {
		return
	}
	switch strings.ToLower(jrt.GetProperty(dpiAutoScaleProperty, "")) {
	case "", "quarter", "exact":
	default:
		ErrorCodeThrowableDetail(ERROR_NOT_IMPLEMENTED, nil, ": monitor-specific scaling is only implemented for swt.autoScale \"quarter\" and \"exact\"")
	}
}

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
