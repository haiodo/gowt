package gowt

import "github.com/haiodo/gowt/swt"

// A job saved as PDF needs no printer on macOS: the print operation saves it.
const pdfPrinterRequired = false

func gcHandle(gc *swt.GC) uintptr { return uintptr(gc.Handle.GraphicsPort()) }
