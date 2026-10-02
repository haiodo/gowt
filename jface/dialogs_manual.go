package jface

import "github.com/haiodo/gowt/swt"

// Hand-written stand-ins for what layout needs of dialogs.Dialog and resource.JFaceResources
// (both are slice B); j2go maps their static calls here through Manual.

type Dialog struct{}

type JFaceResources struct{}

const (
	horizontalDialogUnitPerChar = 4
	verticalDialogUnitsPerChar  = 8
)

func DialogConvertHeightInCharsToPixels(fm *swt.FontMetrics, chars int32) int32 {
	return fm.GetHeight() * chars
}

func DialogConvertHorizontalDLUsToPixels(fm *swt.FontMetrics, dlus int32) int32 {
	return int32((fm.GetAverageCharacterWidth()*float64(dlus) + horizontalDialogUnitPerChar/2) / horizontalDialogUnitPerChar)
}

func DialogConvertVerticalDLUsToPixels(fm *swt.FontMetrics, dlus int32) int32 {
	return (fm.GetHeight()*dlus + verticalDialogUnitsPerChar/2) / verticalDialogUnitsPerChar
}

func DialogConvertWidthInCharsToPixels(fm *swt.FontMetrics, chars int32) int32 {
	return int32(fm.GetAverageCharacterWidth() * float64(chars))
}

// JFace's dialog font is a registry entry that defaults to the system font.
func JFaceResourcesGetDialogFont() *swt.Font {
	d := swt.DisplayGetCurrent()
	if d == nil {
		d = swt.DisplayGetDefault()
	}
	return d.GetSystemFont()
}
