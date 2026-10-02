// Package winmanifest embeds an application manifest into a Windows executable: comctl32 v6
// (visual styles, SysLink) and per-monitor DPI awareness (v2, falling back to per-monitor and
// system-aware on older Windows). Import it for its side effect from the main program:
//
//	import _ "github.com/haiodo/gowt/winmanifest"
//
// The resources are manifest_windows_{amd64,arm64}.syso, made from app.manifest by
// tooling/mksyso (make winmanifest). Elsewhere the package is empty.
package winmanifest
