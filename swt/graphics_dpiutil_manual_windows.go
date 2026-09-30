// DPIUtil as win32 needs it on top of the shared single-zoom stand-in (graphics_stubs_manual.go): the
// process runs at 100% (the OS stretches a DPI-unaware process), so every zoom conversion is the identity.
package swt

const (
	AutoscalingModeDISABLED AutoscalingMode = iota
	AutoscalingModeDISABLED_INHERITED
	AutoscalingModeENABLED
)

func DPIUtilGetZoomForAutoscaleProperty(nativeDeviceZoom int32) int32 { return 100 }
func DPIUtilMapDPIToZoom(dpi int32) int32                             { return dpi * 100 / 96 }
func DPIUtilMapZoomToDPI(zoom int32) int32                            { return zoom * 96 / 100 }
func DPIUtilGetScalingFactor(zoom int32) float32                      { return float32(zoom) / 100 }
func DPIUtilSetMonitorSpecificScaling(active bool)                    {}
func DPIUtilIsMonitorSpecificScalingActive() bool                     { return false }
func DPIUtilIsCustomAutoScale() bool                                  { return false }
