//go:build !darwin

package gowt

import (
	"runtime"

	"github.com/haiodo/gowt/swt"
)

func systemDark() bool { return swt.DisplayIsSystemDarkTheme() }

// secondaryText is COLOR_GRAYTEXT on Windows (SWT's disabled foreground) and the insensitive label
// colour on GTK (SWT's inactive title foreground).
func secondaryText() RGB {
	id := int32(swt.COLOR_TITLE_INACTIVE_FOREGROUND)
	if runtime.GOOS == "windows" {
		id = swt.COLOR_WIDGET_DISABLED_FOREGROUND
	}
	return rgbOf(swt.DisplayGetCurrent().GetSystemColor(id).GetRGB())
}
