package gowt

import "github.com/haiodo/gowt/swt"

// GTK's file backend writes the PDF; no printer is needed.
const pdfPrinterRequired = false

func gcHandle(gc *swt.GC) uintptr { return uintptr(gc.Handle) }
